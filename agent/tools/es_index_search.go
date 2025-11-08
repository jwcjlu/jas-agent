package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"jas-agent/agent/core"
	"strings"
)

// SearchIndices 模糊搜索索引名称
type SearchIndices struct {
	conn *ESConnection
}

func NewSearchIndices(conn *ESConnection) *SearchIndices {
	return &SearchIndices{conn: conn}
}

func (t *SearchIndices) Name() string {
	return "search_indices"
}

func (t *SearchIndices) Description() string {
	return "根据关键词模糊搜索索引名称。输入：搜索关键词（如'log'、'user'、'2024-11'等）。返回：包含该关键词的所有索引列表。当你不确定索引的完整名称时使用此工具。"
}

func (t *SearchIndices) Input() any {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"keyword": map[string]interface{}{
				"type":        "string",
				"description": "搜索关键词，支持部分匹配",
			},
		},
		"required": []string{"keyword"},
	}
}

func (t *SearchIndices) Type() core.ToolType {
	return core.Normal
}

func (t *SearchIndices) Handler(ctx context.Context, input string) (string, error) {
	keyword := strings.ToLower(strings.TrimSpace(input))
	if keyword == "" {
		return "", fmt.Errorf("search keyword is required")
	}

	// 获取所有索引
	respBody, err := t.conn.doRequest(ctx, "GET", "/_cat/indices?v&format=json", nil)
	if err != nil {
		return "", err
	}

	var indices []map[string]interface{}
	if err := json.Unmarshal(respBody, &indices); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(indices) == 0 {
		return "No indices found in cluster", nil
	}

	// 模糊匹配
	var matchedIndices []map[string]interface{}
	for _, index := range indices {
		indexName := fmt.Sprintf("%v", index["index"])
		if strings.Contains(strings.ToLower(indexName), keyword) {
			matchedIndices = append(matchedIndices, index)
		}
	}

	if len(matchedIndices) == 0 {
		return fmt.Sprintf("未找到包含关键词 '%s' 的索引。\n\n建议：使用 list_indices 查看所有索引，或尝试其他关键词。", keyword), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("找到 %d 个包含关键词 '%s' 的索引：\n\n", len(matchedIndices), keyword))

	// 收集所有索引名称用于分析
	var indexNames []string
	for _, index := range matchedIndices {
		indexName := fmt.Sprintf("%v", index["index"])
		indexNames = append(indexNames, indexName)
		docsCount := index["docs.count"]
		storeSize := index["store.size"]
		health := index["health"]

		result.WriteString(fmt.Sprintf("- %s\n", indexName))
		result.WriteString(fmt.Sprintf("  Health: %s, Docs: %v, Size: %v\n", health, docsCount, storeSize))
	}

	// 检测是否有相同前缀的索引（仅日期不同）
	wildcardSuggestion := detectWildcardPattern(indexNames)
	if wildcardSuggestion != "" {
		// 找出最新的索引（按名称排序，通常日期在后面会更大）
		latestIndex := findLatestIndex(indexNames)
		result.WriteString(fmt.Sprintf("\n💡 查询策略建议：\n"))
		result.WriteString(fmt.Sprintf("   1️⃣ 优先查询最新索引：'%s'\n", latestIndex))
		result.WriteString(fmt.Sprintf("   2️⃣ 如果查不到数据，再使用通配符 '%s' 查询所有相关索引\n", wildcardSuggestion))
	}

	return result.String(), nil
}

// detectWildcardPattern 检测是否可以使用通配符模式
func detectWildcardPattern(indexNames []string) string {
	if len(indexNames) < 2 {
		return ""
	}

	// 提取公共前缀
	commonPrefix := findCommonPrefix(indexNames)
	if commonPrefix == "" || len(commonPrefix) < 3 {
		return ""
	}

	// 检查是否所有索引都共享这个前缀，且后缀看起来像日期
	for _, name := range indexNames {
		if !strings.HasPrefix(name, commonPrefix) {
			return ""
		}
		// 检查后缀是否像日期（包含数字、点、横线）
		suffix := strings.TrimPrefix(name, commonPrefix)
		if !isDateLikeSuffix(suffix) {
			return ""
		}
	}

	return commonPrefix + "*"
}

// findCommonPrefix 找到所有字符串的公共前缀
func findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0]
	for _, str := range strs[1:] {
		for i := 0; i < len(prefix) && i < len(str); i++ {
			if prefix[i] != str[i] {
				prefix = prefix[:i]
				break
			}
		}
		if len(prefix) == 0 {
			return ""
		}
	}

	// 去掉末尾的日期分隔符（如 -、_、.）
	prefix = strings.TrimRight(prefix, "-_.")

	return prefix
}

// isDateLikeSuffix 判断后缀是否像日期格式
func isDateLikeSuffix(suffix string) bool {
	if suffix == "" {
		return false
	}
	// 检查是否包含数字和日期分隔符
	hasDigit := false
	for _, ch := range suffix {
		if ch >= '0' && ch <= '9' {
			hasDigit = true
			break
		}
	}
	return hasDigit && (strings.Contains(suffix, "-") || strings.Contains(suffix, ".") || strings.Contains(suffix, "_"))
}

// findLatestIndex 找到最新的索引（按字符串排序，日期通常越新越大）
func findLatestIndex(indexNames []string) string {
	if len(indexNames) == 0 {
		return ""
	}

	latest := indexNames[0]
	for _, name := range indexNames[1:] {
		// 字符串比较，日期格式通常越新的越大
		// 例如: 2025.11.04 > 2025.11.03
		if name > latest {
			latest = name
		}
	}

	return latest
}
