package core

import "context"

type Tool interface {
	Name() string
	Description() string
	Handler(ctx context.Context, input string) (string, error)
}

type FilterFunc func(tool Tool) bool
