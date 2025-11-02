# Chain Framework 示例

这个示例演示了如何使用 JAS Agent 的链式框架（Chain Framework）。

## 什么是 Chain Framework？

Chain Framework 允许你将多个 Agent 按照预定义的流程串联起来，前一个 Agent 的输出会作为下一个 Agent 的输入。这类似于流水线或工作流的概念。

## 运行示例

```bash
cd examples/chain
go run . -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL
```

## 示例说明

### 示例1: 简单线性链

演示最基本的链式执行：查询狗狗体重 -> 计算总和

### 示例2: 带转换的链

演示如何使用转换函数处理节点输出

### 示例3: 条件分支链

演示如何根据结果选择不同的处理路径

## 更多信息

详细的使用指南请参考：[Chain 和 Plan 框架使用指南](../../docs/CHAIN_AND_PLAN_FRAMEWORK.md)

