package core

type Memory interface {
	AddMessage(message Message)
	AddMessages(messages []Message)
	GetLastMessage() Message
	GetMessages() []Message
	GetFormatMessage() string
	Clear()
	GetMessage(rt RoleType) Message
}
