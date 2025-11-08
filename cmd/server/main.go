package main

import (
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"

	"jas-agent/internal/conf"

	_ "jas-agent/agent/examples/react/tools" // 注册工具
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "conf", "configs/config.yaml", "配置文件路径")
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
