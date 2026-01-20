package main

import (
	"context"
	"fmt"
	"time"

	"jas-agent/agent/core"
)

func main() {
	// 创建事件总线
	eventBus := core.NewEventBus()

	// 订阅Agent完成事件
	eventBus.Subscribe(core.EventAgentFinished, func(ctx context.Context, event *core.Event) error {
		payload := event.Payload.(map[string]interface{})
		fmt.Printf("✅ Agent执行完成: ID=%s, 类型=%s, 耗时=%dms, 成功=%v\n",
			payload["agent_id"],
			payload["agent_type"],
			payload["duration_ms"],
			payload["success"])
		return nil
	})

	// 订阅所有事件
	eventBus.SubscribeAll(func(ctx context.Context, event *core.Event) error {
		fmt.Printf("📢 事件: %s, TraceID=%s\n", event.Type, event.TraceID)
		return nil
	})

	// 模拟发布事件
	ctx := context.Background()
	eventBus.Publish(ctx, core.EventAgentStarted, map[string]interface{}{
		"agent_id":   "example_agent",
		"agent_type": "ReactAgent",
		"query":      "示例查询",
	})

	time.Sleep(100 * time.Millisecond)

	eventBus.Publish(ctx, core.EventAgentStepDone, map[string]interface{}{
		"agent_id":    "example_agent",
		"agent_type":  "ReactAgent",
		"step":        1,
		"duration_ms": 500,
	})

	time.Sleep(100 * time.Millisecond)

	eventBus.Publish(ctx, core.EventAgentFinished, map[string]interface{}{
		"agent_id":    "example_agent",
		"agent_type":  "ReactAgent",
		"duration_ms": 1500,
		"total_steps": 2,
		"success":     true,
		"result":      "执行成功",
	})

	time.Sleep(200 * time.Millisecond)
	eventBus.Close()
	fmt.Println("✅ 事件总线示例完成")
}
