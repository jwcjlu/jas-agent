package main

import (
	"context"
	"encoding/json"
	"fmt"

	"jas-agent/agent/core"
)

func main() {
	// 创建状态管理器
	stateManager := core.NewInMemoryStateManager()

	ctx := context.Background()

	// 创建并保存状态快照
	snapshot := &core.StateSnapshot{
		AgentID:     "example_agent_123",
		AgentType:   "ReactAgent",
		State:       "Running",
		CurrentStep: 2,
		MaxSteps:    10,
		Query:       "示例查询",
		Results:     []string{"结果1", "结果2"},
		Metadata: map[string]interface{}{
			"custom_field": "custom_value",
		},
	}

	err := stateManager.Save(ctx, snapshot)
	if err != nil {
		fmt.Printf("保存失败: %v\n", err)
		return
	}
	fmt.Println("✅ 状态快照已保存")

	// 加载状态快照
	loaded, err := stateManager.Load(ctx, "example_agent_123")
	if err != nil {
		fmt.Printf("加载失败: %v\n", err)
		return
	}

	// 打印快照信息
	data, _ := json.MarshalIndent(loaded, "", "  ")
	fmt.Printf("📸 加载的状态快照:\n%s\n", data)

	// 列出所有快照
	allSnapshots, err := stateManager.List(ctx, "ReactAgent")
	if err != nil {
		fmt.Printf("列出失败: %v\n", err)
		return
	}

	fmt.Printf("📋 所有ReactAgent快照数量: %d\n", len(allSnapshots))

	// 删除快照
	err = stateManager.Delete(ctx, "example_agent_123")
	if err != nil {
		fmt.Printf("删除失败: %v\n", err)
		return
	}
	fmt.Println("✅ 状态快照已删除")

	// 验证删除
	_, err = stateManager.Load(ctx, "example_agent_123")
	if err != nil {
		fmt.Println("✅ 快照已成功删除（无法加载）")
	}
}
