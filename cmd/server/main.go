package main

import (
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"

	"jas-agent/agent/core"
	"jas-agent/internal/conf"

	_ "jas-agent/agent/examples/react/tools" // 注册工具
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "conf", "E://configs/jas-agent/config.yaml", "配置文件路径")
	flag.Parse()

	stdLogger := log.NewStdLogger(os.Stdout)
	logger := log.With(stdLogger,
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", "jas-agent",
	)
	helper := log.NewHelper(logger)

	confLoader := config.New(
		config.WithSource(
			file.NewSource(configPath),
		),
	)
	if err := confLoader.Load(); err != nil {
		helper.Fatalf("加载配置失败: %v", err)
	}
	defer confLoader.Close()

	var bootstrap conf.Bootstrap
	if err := confLoader.Scan(&bootstrap); err != nil {
		helper.Fatalf("解析配置失败: %v", err)
	}

	// 初始化可观测性系统（追踪和指标）
	obsCleanup, err := core.InitObservability("jas-agent", "1.0.0")
	if err != nil {
		helper.Warnf("初始化可观测性系统失败（继续运行）: %v", err)
	} else {
		helper.Info("✅ 可观测性系统初始化成功")
		defer func() {
			if obsCleanup != nil {
				obsCleanup()
			}
		}()
	}

	// 设置默认事件监听器（日志、指标、状态快照）
	metrics := core.GetMetrics()
	stateManager := core.GetGlobalStateManager()
	core.SetupDefaultEventListeners(logger, metrics, stateManager)
	helper.Info("✅ 事件监听器设置完成")

	app, cleanup, err := wireApp(&bootstrap, logger)
	if err != nil {
		helper.Fatalf("构建应用失败: %v", err)
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	helper.Info("🚀 启动 JAS Agent 服务器...")
	if err := app.Run(); err != nil {
		helper.Fatalf("服务运行失败: %v", err)
	}
}
