package llm

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
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
		t.Logf("匹配 %d:", i+1)
		t.Logf("  完整: %s", match[0])

		if match[2] != "" {
			// 有模块名的情况
			t.Logf("  模块: %s", match[1])
			t.Logf("  函数: %s", match[2])
		} else {
			// 没有模块名的情况
			t.Logf("  函数: %s", match[1])
		}

		// 解析JSON
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(match[3]), &params); err == nil {
			t.Logf("  参数: %+v", params)
		} else {
			t.Logf("  参数解析错误: %v", err)
		}
		t.Log("")
	}
}

func TestCallTools(tb *testing.T) {
	ctx := context.Background()
	config := openai.DefaultConfig("sk-DKPXsITvLkWVdtN_gTRPdTmv9ILx3VJ1FVy9AI8ur4R9P_3oM4INlil_Or8")

	config.BaseURL = "http://10.86.3.248:3000/v1"

	client := openai.NewClientWithConfig(config)

	// describe the function & its inputs
	params := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"location": {
				Type:        jsonschema.String,
				Description: "The city and state, e.g. San Francisco, CA",
			},
			"unit": {
				Type: jsonschema.String,
				Enum: []string{"celsius", "fahrenheit"},
			},
		},
		Required: []string{"location"},
	}
	f := openai.FunctionDefinition{
		Name:        "get_current_weather",
		Description: "Get the current weather in a given location",
		Parameters:  params,
	}
	t := openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &f,
	}

	// simulate user asking a question that requires the function
	dialogue := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "What is the weather in Boston today?"},
	}
	tb.Logf("Asking OpenAI '%v' and providing it a '%v()' function...",
		dialogue[0].Content, f.Name)
	resp, err := client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: dialogue,
			Tools:    []openai.Tool{t},
		},
	)
	if err != nil || len(resp.Choices) != 1 {
		tb.Logf("Completion error: err:%v len(choices):%v", err,
			len(resp.Choices))
		return
	}
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		tb.Logf("Completion error: len(toolcalls): %v", len(msg.ToolCalls))
		return
	}

	// simulate calling the function & responding to OpenAI
	dialogue = append(dialogue, msg)
	tb.Logf("OpenAI called us back wanting to invoke our function '%v' with params '%v'",
		msg.ToolCalls[0].Function.Name, msg.ToolCalls[0].Function.Arguments)
	dialogue = append(dialogue, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    "Sunny and 80 degrees.",
		Name:       msg.ToolCalls[0].Function.Name,
		ToolCallID: msg.ToolCalls[0].ID,
	})
	tb.Logf("Sending OpenAI our '%v()' function's response and requesting the reply to the original question...",
		f.Name)
	resp, err = client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: dialogue,
			Tools:    []openai.Tool{t},
		},
	)
	if err != nil || len(resp.Choices) != 1 {
		tb.Logf("2nd completion error: err:%v len(choices):%v", err,
			len(resp.Choices))
		return
	}

	// display OpenAI's response to the original question utilizing our function
	msg = resp.Choices[0].Message
	tb.Logf("OpenAI answered the original request with: %v",
		msg.Content)
}
