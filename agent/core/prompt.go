package core

import (
	"fmt"
	"strings"
)

// init 包初始化时注册内置模版
func init() {
	initReactTemplate()
	initSummaryTemplate()
	initSQLTemplate()
	initPlanTemplate()
	initESTemplate()
}

type ToolData struct {
	Name        string `json:"name"`
	Input       any    `json:"input"`
	Description string `json:"description"`
}

type ReactSystemPrompt struct {
	Date  string     `json:"date"`
	Tools []ToolData `json:"tools"`
}

// initReactTemplate 初始化 React 模版
func initReactTemplate() {
	reactTemplate := NewPromptTemplate(
		"react_system",
		"ReAct Agent 系统提示词模版",
		`你是一个基于ReAct框架的AI代理。你必须严格按照思考-行动-观察的循环解决问题。

					当前时间: {{.Date}}
					
					可用工具:
					{{.Tools}}
					
					重要约束条件:
						1. 每次只能执行一个步骤，不要一次性生成所有步骤
						2. 每次响应必须只包含一个思考和一个行动
						3. 思考格式: Thought: [你的思考过程]
						4. 行动格式: Action: toolName[input] 或 Action: Finish[final answer]
						5. 等待观察结果后再进行下一步思考
						6. 不要预测工具执行结果，等待实际观察
					
					{{.Examples}}
					
					请开始第一步思考。`,
	).AddVariable("Date", "当前时间").
		AddVariable("Tools", "可用工具列表").
		AddVariable("Examples", "Few-shot 示例").
		AddExample(
			"计算 15 + 27 的结果",
			`Thought: 我需要计算 15 + 27 的结果
                    Action: calculator[15 + 27]`,
			"数学计算任务",
		).
		AddExample(
			"查询边境牧羊犬的平均体重",
			`Thought: 我需要查询边境牧羊犬的平均体重
                    Action: averageDogWeight[border collie]`,
			"信息查询任务",
		)

	RegisterGlobalTemplate(reactTemplate)
}

// GetReactSystemPrompt 生成React系统提示词
func GetReactSystemPrompt(prompt ReactSystemPrompt) string {
	// 构建工具描述
	var toolsDesc strings.Builder
	for _, tool := range prompt.Tools {
		toolsDesc.WriteString(fmt.Sprintf("- %s: %s", tool.Name, tool.Description))
		if tool.Input != nil {
			toolsDesc.WriteString(fmt.Sprintf(" :%v", tool.Input))
		}
		toolsDesc.WriteString("\n")
	}

	// 使用模版构建提示词
	data := map[string]interface{}{
		"Date":  prompt.Date,
		"Tools": toolsDesc.String(),
	}

	result, err := BuildGlobalPrompt("react_system", data)
	if err != nil {
		// 如果模版构建失败，回退到原始实现
		return fmt.Sprintf(`你是一个基于ReAct框架的AI代理。你必须严格按照思考-行动-观察的循环解决问题。

					当前时间: %s
					
					可用工具:
					%s
					
					重要约束条件:
						1. 每次只能执行一个步骤，不要一次性生成所有步骤
                        2. 每次响应必须只包含一个思考和一个行动
						3. 思考格式: Thought: [你的思考过程]
						4. 行动格式: Action: toolName[input] 或 Action: Finish[final answer]
						5. 等待观察结果后再进行下一步思考
						6. 不要预测工具执行结果，等待实际观察
					
					请开始第一步思考。`, prompt.Date, toolsDesc.String())
	}

	return result
}

// initSummaryTemplate 初始化 Summary 模版
func initSummaryTemplate() {
	summaryTemplate := NewPromptTemplate(
		"summary_system",
		"Summary Agent 系统提示词模版",
		`你是一个专业的信息处理助理，擅长总结和结构化展示内容。

				总结要求:
					1. 分析整个执行过程
					2. 提取关键信息和结果
					3. 提供简洁明了的最终答案
					4. 确保答案准确无误
                    5. 如果总结内容中包含任何列表性质的信息（如产品特性、功能对比、步骤、优缺点、数据清单等），请自动使用Markdown表格来呈现，而不是使用项目符号
					6. 严禁使用未列出的工具名称（如 search、browse 等）。只能从"可用工具"列表中选择。
				
				{{.Examples}}
				
				请基于执行过程提供总结。`,
	).AddVariable("Examples", "Few-shot 示例").
		AddExample(
			"计算任务执行过程",
			"基于执行过程分析，15加27等于42，乘以3等于126。因此结果是126。",
			"数学计算完成后",
		).
		AddExample(
			"信息查询执行过程",
			"根据查询结果，边境牧羊犬的平均体重约为37磅（约17公斤）。",
			"信息查询完成后",
		)

	RegisterGlobalTemplate(summaryTemplate)
}

// GetSummarySystemPrompt 生成总结系统提示词
func GetSummarySystemPrompt() string {
	// 使用模版构建提示词
	data := map[string]interface{}{}

	result, err := BuildGlobalPrompt("summary_system", data)
	if err != nil {
		// 如果模版构建失败，回退到原始实现
		return `你是一个专业的总结助手。你的任务是对执行过程进行总结，提供清晰、准确的最终答案。

				总结要求:
					1. 分析整个执行过程
					2. 提取关键信息和结果
					3. 提供简洁明了的最终答案
					4. 确保答案准确无误
					5. 严禁使用未列出的工具名称（如 search、browse 等）。只能从"可用工具"列表中选择。
				
				请基于执行过程提供总结。`
	}

	return result
}

type SQLSystemPrompt struct {
	Date         string     `json:"date"`
	Tools        []ToolData `json:"tools"`
	DatabaseInfo string     `json:"database_info"`
}

type ESSystemPrompt struct {
	Date        string     `json:"date"`
	Tools       []ToolData `json:"tools"`
	ClusterInfo string     `json:"cluster_info"`
}

// initSQLTemplate 初始化 SQL 模版
func initSQLTemplate() {
	sqlTemplate := NewPromptTemplate(
		"sql_system",
		"SQL Agent 系统提示词模版",
		`你是一个专业的SQL查询助手。你的核心职责是根据用户需求生成准确、高效的SQL查询。

			当前时间: {{.Date}}
			数据库信息: {{.DatabaseInfo}}
			
			可用工具:
			{{.Tools}}
			
			工作流程:
				1. **理解需求**: 仔细分析用户的查询需求
				2. **探索Schema**: 使用 list_tables 了解数据库结构，使用 tables_schema 获取表详情
				3. **编写SQL**: 基于Schema信息编写准确的SQL查询
				4. **执行查询**: 使用 execute_sql 执行查询
				5. **分析结果**: 解读查询结果，回答用户问题
			
			SQL编写规范:
				- 使用标准SQL语法
				- 仅使用 SELECT 查询（安全限制）
				- 正确使用 JOIN 关联多表
				- 适当使用聚合函数（COUNT, SUM, AVG等）
				- 添加必要的 WHERE 条件过滤
				- 使用 ORDER BY 排序结果
				- 使用 LIMIT 限制结果数量（避免返回过多数据）
			
			重要约束:
				1. 每次只执行一个步骤
				2. 必须先了解Schema再编写SQL
				3. SQL语句必须基于实际的表结构
				4. 思考格式: Thought: [你的思考过程]
				5. 行动格式: Action: toolName[input] 或 Action: Finish[final answer]
				6. 等待观察结果后再进行下一步
			
			{{.Examples}}
			
			请开始帮助用户完成SQL查询任务。`,
	).AddVariable("Date", "当前时间").
		AddVariable("DatabaseInfo", "数据库信息").
		AddVariable("Tools", "可用工具列表").
		AddVariable("Examples", "Few-shot 示例").
		AddExample(
			"查询：列出数据库中的所有表",
			`Thought: 我需要先了解数据库中有哪些表
                    Action: list_tables[]`,
			"探索数据库结构",
		).
		AddExample(
			"查询：查询用户表有多少条记录",
			`Thought: 我需要先查看用户表的结构，然后计算记录数
                    Action: tables_schema[users]
                    Action: execute_sql[SELECT COUNT(*) FROM users]`,
			"统计查询任务",
		).
		AddExample(
			"查询：查询每个用户的订单总金额",
			`Thought: 我需要了解用户表和订单表的结构，然后编写关联查询
                   Action: tables_schema[users,orders]
                   Action: execute_sql[SELECT u.username, SUM(o.amount) as total FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.id ORDER BY total DESC]`,
			"复杂关联查询任务",
		)

	RegisterGlobalTemplate(sqlTemplate)
}

// GetSQLSystemPrompt 生成SQL Agent系统提示词
func GetSQLSystemPrompt(prompt SQLSystemPrompt) string {
	// 构建工具描述
	var toolsDesc strings.Builder
	for _, tool := range prompt.Tools {
		toolsDesc.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}

	// 使用模版构建提示词
	data := map[string]interface{}{
		"Date":         prompt.Date,
		"DatabaseInfo": prompt.DatabaseInfo,
		"Tools":        toolsDesc.String(),
	}

	result, err := BuildGlobalPrompt("sql_system", data)
	if err != nil {
		// 如果模版构建失败，回退到原始实现
		return fmt.Sprintf(`你是一个专业的SQL查询助手。你的核心职责是根据用户需求生成准确、高效的SQL查询。

           当前时间: %s
           数据库信息: %s
           
           可用工具:
           %s
           
           工作流程:
			   1. **理解需求**: 仔细分析用户的查询需求
			   2. **探索Schema**: 使用 list_tables 了解数据库结构，使用 tables_schema 获取表详情
			   3. **编写SQL**: 基于Schema信息编写准确的SQL查询
			   4. **执行查询**: 使用 execute_sql 执行查询
			   5. **分析结果**: 解读查询结果，回答用户问题
           
           SQL编写规范:
			   - 使用标准SQL语法
			   - 仅使用 SELECT 查询（安全限制）
			   - 正确使用 JOIN 关联多表
			   - 适当使用聚合函数（COUNT, SUM, AVG等）
			   - 添加必要的 WHERE 条件过滤
			   - 使用 ORDER BY 排序结果
			   - 使用 LIMIT 限制结果数量（避免返回过多数据）
           
           重要约束:
			   1. 每次只执行一个步骤
			   2. 必须先了解Schema再编写SQL
			   3. SQL语句必须基于实际的表结构
			   4. 思考格式: Thought: [你的思考过程]
			   5. 行动格式: Action: toolName[input] 或 Action: Finish[final answer]
			   6. 等待观察结果后再进行下一步
           
           请开始帮助用户完成SQL查询任务。`, prompt.Date, prompt.DatabaseInfo, toolsDesc.String())
	}

	return result
}

// initPlanTemplate 初始化 Plan 模版
func initPlanTemplate() {
	planTemplate := NewPromptTemplate(
		"plan_system",
		"Plan Agent 系统提示词模版",
		`你是一个专业的任务规划助手。你的职责是将复杂任务分解为可执行的步骤序列。

规划原则:
	1. 分析任务，识别所有必要的子任务
	2. 确定任务之间的依赖关系
	3. 选择合适的工具完成每个步骤
	4. 生成清晰、可执行的计划
	5. 考虑异常情况和备选方案

计划格式要求:
	- 使用JSON格式输出
	- 每个步骤必须具体、可操作
	- 明确标注步骤间的依赖关系
	- 合理安排执行顺序

执行特点:
	- 先规划，后执行
	- 可根据执行结果动态调整计划
	- 每步执行完成后记录结果
	- 支持并行执行无依赖的步骤

{{.Examples}}

请基于用户任务生成详细的执行计划。`,
	).AddVariable("Examples", "Few-shot 示例").
		AddExample(
			"复杂计算任务",
			`{
  "goal": "计算三只狗的总体重",
  "steps": [
    {"id": 1, "description": "查询边境牧羊犬体重", "tool": "averageDogWeight", "input": "border collie", "dependencies": []},
    {"id": 2, "description": "查询苏格兰梗体重", "tool": "averageDogWeight", "input": "scottish terrier", "dependencies": []},
    {"id": 3, "description": "查询玩具贵宾犬体重", "tool": "averageDogWeight", "input": "toy poodle", "dependencies": []},
    {"id": 4, "description": "计算总重量", "tool": "calculator", "input": "${step.1} + ${step.2} + ${step.3}", "dependencies": [1, 2, 3]}
  ]
}`,
			"多步骤计算任务",
		).
		AddExample(
			"SQL查询任务",
			`{
  "goal": "查询销售数据统计",
  "steps": [
    {"id": 1, "description": "列出所有表", "tool": "list_tables", "input": "", "dependencies": []},
    {"id": 2, "description": "查看订单表结构", "tool": "tables_schema", "input": "orders", "dependencies": [1]},
    {"id": 3, "description": "执行统计查询", "tool": "execute_sql", "input": "SELECT DATE(order_date), SUM(amount) FROM orders GROUP BY DATE(order_date)", "dependencies": [2]}
  ]
}`,
			"数据库查询任务",
		)

	RegisterGlobalTemplate(planTemplate)
}

// GetPlanSystemPrompt 生成Plan系统提示词
func GetPlanSystemPrompt() string {
	// 使用模版构建提示词
	data := map[string]interface{}{}

	result, err := BuildGlobalPrompt("plan_system", data)
	if err != nil {
		// 如果模版构建失败，回退到原始实现
		return `你是一个专业的任务规划助手。你的职责是将复杂任务分解为可执行的步骤序列。

规划原则:
	1. 分析任务，识别所有必要的子任务
	2. 确定任务之间的依赖关系
	3. 选择合适的工具完成每个步骤
	4. 生成清晰、可执行的计划
	5. 考虑异常情况和备选方案

计划格式要求:
	- 使用JSON格式输出
	- 每个步骤必须具体、可操作
	- 明确标注步骤间的依赖关系
	- 合理安排执行顺序

请基于用户任务生成详细的执行计划。`
	}

	return result
}
