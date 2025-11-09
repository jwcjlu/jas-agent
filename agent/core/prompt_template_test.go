package core

import (
	"testing"
)

func TestPromptTemplate(t *testing.T) {
	// 创建测试模版
	template := NewPromptTemplate(
		"test_template",
		"测试模版",
		`你好 {{.Name}}！

					当前时间: {{.Date}}
					任务: {{.Task}}
					
					{{.Examples}}
					
					请开始执行任务。`,
	).AddVariable("Name", "用户名称").
		AddVariable("Date", "当前时间").
		AddVariable("Task", "任务描述").
		AddVariable("Examples", "Few-shot 示例").
		AddExample(
			"计算 1 + 1",
			"结果是 2",
			"简单计算",
		).
		AddExample(
			"查询天气",
			"今天天气晴朗",
			"信息查询",
		)

	// 测试模版构建
	data := map[string]interface{}{
		"Name": "张三",
		"Date": "2024-01-01 12:00:00",
		"Task": "完成数学计算",
	}

	result, err := template.Build(data)
	if err != nil {
		t.Fatalf("模版构建失败: %v", err)
	}

	// 验证结果包含预期内容
	expectedContents := []string{
		"你好 张三！",
		"当前时间: 2024-01-01 12:00:00",
		"任务: 完成数学计算",
		"示例:",
		"示例 1:",
		"输入: 计算 1 + 1",
		"输出: 结果是 2",
		"上下文: 简单计算",
	}

	for _, content := range expectedContents {
		if !contains(result, content) {
			t.Errorf("结果中缺少预期内容: %s", content)
		}
	}

	t.Logf("生成的提示词:\n%s", result)
}

func TestPromptManager(t *testing.T) {
	manager := NewPromptManager()

	// 创建测试模版
	template := NewPromptTemplate(
		"manager_test",
		"管理器测试模版",
		"测试内容: {{.Test}}",
	).AddVariable("Test", "测试变量")

	// 注册模版
	manager.RegisterTemplate(template)

	// 测试获取模版
	retrievedTemplate, err := manager.GetTemplate("manager_test")
	if err != nil {
		t.Fatalf("获取模版失败: %v", err)
	}

	if retrievedTemplate.Name != "manager_test" {
		t.Errorf("模版名称不匹配: 期望 %s, 实际 %s", "manager_test", retrievedTemplate.Name)
	}

	// 测试构建提示词
	data := map[string]interface{}{
		"Test": "成功",
	}

	prompt, err := manager.BuildPrompt("manager_test", data)
	if err != nil {
		t.Fatalf("构建提示词失败: %v", err)
	}

	if !contains(prompt, "测试内容: 成功") {
		t.Errorf("提示词内容不正确: %s", prompt)
	}

	// 测试列出模版
	templates := manager.ListTemplates()
	if len(templates) != 1 || templates[0] != "manager_test" {
		t.Errorf("模版列表不正确: %v", templates)
	}
}

func TestGlobalPromptManager(t *testing.T) {
	// 创建测试模版
	template := NewPromptTemplate(
		"global_test",
		"全局测试模版",
		"全局测试: {{.Message}}",
	).AddVariable("Message", "消息内容")

	// 注册全局模版
	RegisterGlobalTemplate(template)

	// 测试全局构建
	data := map[string]interface{}{
		"Message": "Hello World",
	}

	prompt, err := BuildGlobalPrompt("global_test", data)
	if err != nil {
		t.Fatalf("全局构建失败: %v", err)
	}

	if !contains(prompt, "全局测试: Hello World") {
		t.Errorf("全局提示词内容不正确: %s", prompt)
	}

	// 测试获取全局模版
	globalTemplate, err := GetPromptManager().GetTemplate("global_test")
	if err != nil {
		t.Fatalf("获取全局模版失败: %v", err)
	}

	if globalTemplate.Name != "global_test" {
		t.Errorf("全局模版名称不匹配: 期望 %s, 实际 %s", "global_test", globalTemplate.Name)
	}
}

func TestFewShotExamples(t *testing.T) {
	template := NewPromptTemplate(
		"fewshot_test",
		"Few-shot 测试模版",
		"任务: {{.Task}}\n\n{{.Examples}}",
	).AddVariable("Task", "任务描述").
		AddVariable("Examples", "Few-shot 示例").
		AddExample(
			"问题1",
			"答案1",
			"上下文1",
		).
		AddExample(
			"问题2",
			"答案2",
			"上下文2",
		)

	data := map[string]interface{}{
		"Task": "测试任务",
	}

	result, err := template.Build(data)
	if err != nil {
		t.Fatalf("Few-shot 模版构建失败: %v", err)
	}

	// 验证 Few-shot 示例
	expectedExamples := []string{
		"示例:",
		"示例 1:",
		"输入: 问题1",
		"输出: 答案1",
		"上下文: 上下文1",
		"示例 2:",
		"输入: 问题2",
		"输出: 答案2",
		"上下文: 上下文2",
	}

	for _, content := range expectedExamples {
		if !contains(result, content) {
			t.Errorf("Few-shot 示例中缺少预期内容: %s", content)
		}
	}

	t.Logf("Few-shot 提示词:\n%s", result)
}

func TestTemplateErrorHandling(t *testing.T) {
	manager := NewPromptManager()

	// 测试获取不存在的模版
	_, err := manager.GetTemplate("nonexistent")
	if err == nil {
		t.Error("应该返回错误，但没有")
	}

	// 测试构建不存在的模版
	_, err = manager.BuildPrompt("nonexistent", map[string]interface{}{})
	if err == nil {
		t.Error("应该返回错误，但没有")
	}

	// 测试无效的模版语法
	invalidTemplate := NewPromptTemplate(
		"invalid",
		"无效模版",
		"无效语法: {{.InvalidField",
	)

	_, err = invalidTemplate.Build(map[string]interface{}{})
	if err == nil {
		t.Error("应该返回模版解析错误，但没有")
	}
}

func TestBuiltinTemplates(t *testing.T) {
	// 测试 React 模版
	reactData := map[string]interface{}{
		"Date":  "2024-01-01 12:00:00",
		"Tools": "- calculator: 计算工具",
	}

	reactPrompt, err := BuildGlobalPrompt("react_system", reactData)
	if err != nil {
		t.Fatalf("React 模版构建失败: %v", err)
	}

	if !contains(reactPrompt, "ReAct框架") {
		t.Error("React 模版内容不正确")
	}

	// 测试 Summary 模版
	summaryPrompt, err := BuildGlobalPrompt("summary_system", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Summary 模版构建失败: %v", err)
	}

	if !contains(summaryPrompt, "总结助手") {
		t.Error("Summary 模版内容不正确")
	}

	// 测试 SQL 模版
	sqlData := map[string]interface{}{
		"Date":         "2024-01-01 12:00:00",
		"DatabaseInfo": "MySQL Database: testdb",
		"Tools":        "- list_tables: 列出表",
	}

	sqlPrompt, err := BuildGlobalPrompt("sql_system", sqlData)
	if err != nil {
		t.Fatalf("SQL 模版构建失败: %v", err)
	}

	if !contains(sqlPrompt, "SQL查询助手") {
		t.Error("SQL 模版内容不正确")
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
