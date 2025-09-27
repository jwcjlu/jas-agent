package core

type Message struct {
	Role       RoleType `json:"role"`
	Content    string   `json:"content,omitempty"`
	Name       string   `json:"name,omitempty"`
	ToolCallID string   `json:"tool_call_id,omitempty"`
	ToolCall   []Tool   `json:"tool_call"`
}

type RoleType string

const (
	MessageRoleSystem    = "system"
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleFunction  = "function"
	MessageRoleTool      = "tool"
	MessageRoleDeveloper = "developer"
)
