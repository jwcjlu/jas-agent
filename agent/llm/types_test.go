package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
)

func TestParseToolCall(t *testing.T) {
	text := `测试文本：
    Action: vm_manager@get_index_info[{"area_type": 4103, "siv": 126}]
    Action: get_index_info[{"name": "test", "value": 123}]
    Action: user@login[{"username": "admin"}]`

	// 使用改进的正则表达式
	re := regexp.MustCompile(`Action:\s*(\w+@?\w*)\[({[^]]+})\]`)

	matches := re.FindAllStringSubmatch(text, -1)

	for i, match := range matches {
		fmt.Printf("匹配 %d:\n", i+1)
		fmt.Printf("  完整: %s\n", match[0])

		if match[2] != "" {
			// 有模块名的情况
			fmt.Printf("  模块: %s\n", match[1])
			fmt.Printf("  函数: %s\n", match[2])
		} else {
			// 没有模块名的情况
			fmt.Printf("  函数: %s\n", match[1])
		}

		// 解析JSON
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(match[3]), &params); err == nil {
			fmt.Printf("  参数: %+v\n", params)
		} else {
			fmt.Printf("  参数解析错误: %v\n", err)
		}
		fmt.Println()
	}
}
