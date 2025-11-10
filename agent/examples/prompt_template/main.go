package main

import (
	"os"
	"time"

	"jas-agent/agent/core"

	"github.com/go-kratos/kratos/v2/log"
)

var logger = log.NewHelper(log.With(log.NewStdLogger(os.Stdout), "module", "examples/prompt_template"))

func main() {
	logger.Info("=== JAS Agent Prompt Template 示例 ===")

	// 1. 创建自定义模版
	logger.Info("1. 创建自定义模版")
	customTemplate := core.NewPromptTemplate(
		"custom_math",
		"数学计算助手模版",
		`你是一个数学计算助手。

当前时间: {{.Date}}
用户问题: {{.Question}}

计算规则:
- 支持基本四则运算
- 支持括号优先级
- 结果保留2位小数

{{.Examples}}

请开始计算。`,
	).AddVariable("Date", "当前时间").
		AddVariable("Question", "用户问题").
		AddVariable("Examples", "Few-shot 示例").
		AddExample(
			"计算 2 + 3 * 4",
			`Thought: 根据运算优先级，先算乘法 3*4=12，再算加法 2+12=14
Action: calculator[2 + 3 * 4]
Observation: 14.00
Final Answer: 14.00`,
			"基本运算任务",
		).
		AddExample(
			"计算 (10 + 5) / 3",
			`Thought: 先算括号内 10+5=15，再算除法 15/3=5
Action: calculator[(10 + 5) / 3]
Observation: 5.00
Final Answer: 5.00`,
			"括号运算任务",
		)

	// 注册模版
	core.RegisterGlobalTemplate(customTemplate)

	// 2. 使用模版构建提示词
	logger.Info("2. 使用模版构建提示词")
	data := map[string]interface{}{
		"Date":     time.Now().Format("2006-01-02 15:04:05"),
		"Question": "计算 15 + 27 * 2 的结果",
	}

	prompt, err := core.BuildGlobalPrompt("custom_math", data)
	if err != nil {
		logger.Errorf("错误: %v", err)
		return
	}

	logger.Info("生成的提示词:")
	logger.Info(prompt)
	logger.Info("")

	// 3. 测试内置模版
	logger.Info("3. 测试内置模版")

	// React Agent 模版
	logger.Info("3.1 React Agent 模版:")
	reactData := map[string]interface{}{
		"Date":  time.Now().Format("2006-01-02 15:04:05"),
		"Tools": "- calculator: 数学计算工具\n- averageDogWeight: 查询狗狗平均体重",
	}

	reactPrompt, err := core.BuildGlobalPrompt("react_system", reactData)
	if err != nil {
		logger.Errorf("错误: %v", err)
	} else {
		logger.Info(reactPrompt[:200] + "...")
	}
	logger.Info("")

	// Summary Agent 模版
	logger.Info("3.2 Summary Agent 模版:")
	summaryPrompt, err := core.BuildGlobalPrompt("summary_system", map[string]interface{}{})
	if err != nil {
		logger.Errorf("错误: %v", err)
	} else {
		logger.Info(summaryPrompt[:200] + "...")
	}
	logger.Info("")

	// SQL Agent 模版
	logger.Info("3.3 SQL Agent 模版:")
	sqlData := map[string]interface{}{
		"Date":         time.Now().Format("2006-01-02 15:04:05"),
		"DatabaseInfo": "MySQL Database: testdb",
		"Tools":        "- list_tables: 列出所有表\n- tables_schema: 获取表结构\n- execute_sql: 执行SQL查询",
	}

	sqlPrompt, err := core.BuildGlobalPrompt("sql_system", sqlData)
	if err != nil {
		logger.Errorf("错误: %v", err)
	} else {
		logger.Info(sqlPrompt[:200] + "...")
	}
	logger.Info("")

	// 4. 列出所有可用模版
	logger.Info("4. 列出所有可用模版:")
	templates := core.GetPromptManager().ListTemplates()
	for _, name := range templates {
		logger.Infof("- %s", name)
	}
	logger.Info("")

	// 5. 测试模版变量
	logger.Info("5. 测试模版变量")
	template, err := core.GetPromptManager().GetTemplate("custom_math")
	if err != nil {
		logger.Errorf("错误: %v", err)
		return
	}

	logger.Infof("模版名称: %s", template.Name)
	logger.Infof("模版描述: %s", template.Description)
	logger.Infof("模版变量: %v", template.Variables)
	logger.Infof("Few-shot 示例数量: %d", len(template.Examples))

	for i, example := range template.Examples {
		logger.Infof("  示例 %d:", i+1)
		logger.Infof("    输入: %s", example.Input)
		logger.Infof("    输出: %s", example.Output[:50]+"...")
		logger.Infof("    上下文: %s", example.Context)
	}
}
