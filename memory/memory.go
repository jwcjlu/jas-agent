package memory

import (
	"bytes"
	"fmt"
	"jas-agent/core"
)

func NewMemory() core.Memory {
	return &memory{}
}

type memory struct {
	messages []core.Message
}

func (m *memory) AddMessage(message core.Message) {
	m.messages = append(m.messages, message)
}
func (m *memory) AddMessages(messages []core.Message) {
	for _, message := range messages {
		m.AddMessage(message)
	}
}
func (m *memory) GetLastMessage() core.Message {
	return m.messages[len(m.messages)-1]
}
func (m *memory) GetFormatMessage() string {
	bs := bytes.NewBufferString("")
	for _, message := range m.messages {
		bs.WriteString(fmt.Sprintf("role:%s content:%s\n", message.Role, message.Content))
	}
	return bs.String()
}
func (m *memory) Clear() {
	m.messages = nil
}
func (m *memory) GetMessages() []core.Message {
	return m.messages
}
