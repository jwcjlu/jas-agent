package memory

import (
	"bytes"
	"fmt"
	"jas-agent/core"
)

func NewInMemory() core.Memory {
	return &inMemory{}
}

type inMemory struct {
	messages []core.Message
}

func (m *inMemory) AddMessage(message core.Message) {
	m.messages = append(m.messages, message)
}
func (m *inMemory) AddMessages(messages []core.Message) {
	for _, message := range messages {
		m.AddMessage(message)
	}
}
func (m *inMemory) GetLastMessage() core.Message {
	return m.messages[len(m.messages)-1]
}
func (m *inMemory) GetFormatMessage() string {
	bs := bytes.NewBufferString("")
	for _, message := range m.messages {
		bs.WriteString(fmt.Sprintf("role:%s content:%s\n", message.Role, message.Content))
	}
	return bs.String()
}
func (m *inMemory) Clear() {
	m.messages = nil
}
func (m *inMemory) GetMessages() []core.Message {
	return m.messages
}
