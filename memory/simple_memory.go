package memory

import (
	"bytes"
	"fmt"
	"jas-agent/core"
)

func NewMemory() core.Memory {
	return &simpleMemory{}
}

type simpleMemory struct {
	messages []core.Message
}

func (m *simpleMemory) AddMessage(message core.Message) {
	m.messages = append(m.messages, message)
}
func (m *simpleMemory) AddMessages(messages []core.Message) {
	for _, message := range messages {
		m.AddMessage(message)
	}
}
func (m *simpleMemory) GetLastMessage() core.Message {
	return m.messages[len(m.messages)-1]
}
func (m *simpleMemory) GetFormatMessage() string {
	bs := bytes.NewBufferString("")
	for _, message := range m.messages {
		bs.WriteString(fmt.Sprintf("role:%s content:%s\n", message.Role, message.Content))
	}
	return bs.String()
}
func (m *simpleMemory) Clear() {
	m.messages = nil
}
func (m *simpleMemory) GetMessages() []core.Message {
	return m.messages
}
