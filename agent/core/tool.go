package core

import "context"

type Tool interface {
	Name() string
	Description() string
	Handler(ctx context.Context, input string) (string, error)
	Input() any
	Type() ToolType
}

type ToolType int

const (
	Normal ToolType = 1
	Mcp    ToolType = 2
)

type FilterFunc func(tool Tool) bool
