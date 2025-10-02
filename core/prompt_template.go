package core

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// PromptTemplate 定义提示词模版
type PromptTemplate struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Template    string            `json:"template"`
	Variables   map[string]string `json:"variables"`
	Examples    []FewShotExample  `json:"examples"`
}

// FewShotExample 定义 few-shot 示例
type FewShotExample struct {
	Input     string            `json:"input"`
	Output    string            `json:"output"`
	Context   string            `json:"context,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// PromptBuilder 提示词构建器
type PromptBuilder struct {
	template *PromptTemplate
	data     map[string]interface{}
}

// NewPromptTemplate 创建新的提示词模版
func NewPromptTemplate(name, description, templateStr string) *PromptTemplate {
	return &PromptTemplate{
		Name:        name,
		Description: description,
		Template:    templateStr,
		Variables:   make(map[string]string),
		Examples:    make([]FewShotExample, 0),
	}
}

// AddVariable 添加模版变量
func (pt *PromptTemplate) AddVariable(name, description string) *PromptTemplate {
	pt.Variables[name] = description
	return pt
}

// AddExample 添加 few-shot 示例
func (pt *PromptTemplate) AddExample(input, output, context string) *PromptTemplate {
	example := FewShotExample{
		Input:   input,
		Output:  output,
		Context: context,
	}
	pt.Examples = append(pt.Examples, example)
	return pt
}

// AddExampleWithVariables 添加带变量的 few-shot 示例
func (pt *PromptTemplate) AddExampleWithVariables(input, output, context string, variables map[string]string) *PromptTemplate {
	example := FewShotExample{
		Input:     input,
		Output:    output,
		Context:   context,
		Variables: variables,
	}
	pt.Examples = append(pt.Examples, example)
	return pt
}

// Build 构建提示词
func (pt *PromptTemplate) Build(data map[string]interface{}) (string, error) {
	// 解析模版
	tmpl, err := template.New(pt.Name).Parse(pt.Template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// 准备数据
	templateData := make(map[string]interface{})
	for k, v := range data {
		templateData[k] = v
	}

	// 添加 few-shot 示例
	if len(pt.Examples) > 0 {
		templateData["Examples"] = pt.buildExamples(data)
	}

	// 执行模版
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// buildExamples 构建 few-shot 示例
func (pt *PromptTemplate) buildExamples(data map[string]interface{}) string {
	if len(pt.Examples) == 0 {
		return ""
	}

	var examples strings.Builder
	examples.WriteString("\n\n示例:\n")

	for i, example := range pt.Examples {
		examples.WriteString(fmt.Sprintf("\n示例 %d:\n", i+1))

		if example.Context != "" {
			examples.WriteString(fmt.Sprintf("上下文: %s\n", example.Context))
		}

		examples.WriteString(fmt.Sprintf("输入: %s\n", example.Input))
		examples.WriteString(fmt.Sprintf("输出: %s\n", example.Output))
	}

	return examples.String()
}

// PromptManager 提示词管理器
type PromptManager struct {
	templates map[string]*PromptTemplate
}

// NewPromptManager 创建提示词管理器
func NewPromptManager() *PromptManager {
	return &PromptManager{
		templates: make(map[string]*PromptTemplate),
	}
}

// RegisterTemplate 注册提示词模版
func (pm *PromptManager) RegisterTemplate(template *PromptTemplate) {
	pm.templates[template.Name] = template
}

// GetTemplate 获取提示词模版
func (pm *PromptManager) GetTemplate(name string) (*PromptTemplate, error) {
	template, exists := pm.templates[name]
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", name)
	}
	return template, nil
}

// BuildPrompt 构建提示词
func (pm *PromptManager) BuildPrompt(templateName string, data map[string]interface{}) (string, error) {
	template, err := pm.GetTemplate(templateName)
	if err != nil {
		return "", err
	}
	return template.Build(data)
}

// ListTemplates 列出所有模版
func (pm *PromptManager) ListTemplates() []string {
	var names []string
	for name := range pm.templates {
		names = append(names, name)
	}
	return names
}

// 全局提示词管理器实例
var globalPromptManager = NewPromptManager()

// GetPromptManager 获取全局提示词管理器
func GetPromptManager() *PromptManager {
	return globalPromptManager
}

// RegisterGlobalTemplate 注册全局提示词模版
func RegisterGlobalTemplate(template *PromptTemplate) {
	globalPromptManager.RegisterTemplate(template)
}

// BuildGlobalPrompt 构建全局提示词
func BuildGlobalPrompt(templateName string, data map[string]interface{}) (string, error) {
	return globalPromptManager.BuildPrompt(templateName, data)
}
