# Plan Framework 示例

这个示例演示了如何使用 JAS Agent 的计划框架（Plan Framework）。

## 什么是 Plan Framework？

Plan Framework 采用"先规划，后执行"的策略。它会首先分析任务，生成完整的执行计划，然后按照计划逐步执行。

## 运行示例

```bash
cd examples/plan
go run . -apiKey YOUR_API_KEY -baseUrl YOUR_BASE_URL
```

## 示例说明

### 示例1: 基本计划执行

演示简单的多步骤任务自动规划和执行

### 示例2: 带依赖的复杂计划

演示如何处理有依赖关系的复杂任务

### 示例3: 启用重新规划

演示遇到问题时如何自动调整计划

## 执行流程示例

```
📋 Generating execution plan...

📝 Generated Plan:
Goal: 计算三只狗的总体重
Steps:
  1. 查询边境牧羊犬体重
  2. 查询苏格兰梗体重
  3. 查询玩具贵宾犬体重
  4. 计算总重量 (depends on: [1, 2, 3])

⚙️  Executing step 1: 查询边境牧羊犬体重
✅ Step 1 completed: 37 lbs

⚙️  Executing step 2: 查询苏格兰梗体重
✅ Step 2 completed: 20 lbs

⚙️  Executing step 3: 查询玩具贵宾犬体重
✅ Step 3 completed: 7 lbs

⚙️  Executing step 4: 计算总重量
✅ Step 4 completed: 64

📊 Generating summary...
三只狗的总体重约为64磅。
```

## 更多信息

详细的使用指南请参考：[Chain 和 Plan 框架使用指南](../../../docs/CHAIN_AND_PLAN_FRAMEWORK.md)


