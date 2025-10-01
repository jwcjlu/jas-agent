package tools

import (
	"context"
	"fmt"
	"go.starlark.net/lib/math"
	"go.starlark.net/starlark"
	"jas-agent/core"
)

type Calculator struct {
}

func init() {
	GetToolManager().RegisterTool(&Calculator{})
}

// Description returns a string describing the calculator tool.
func (c *Calculator) Description() string {
	return `Useful for getting the result of a math expression. 
	The input to this tool should be a valid mathematical expression that could be executed by a starlark evaluator.`
}

// Name returns the name of the tool.
func (c *Calculator) Name() string {
	return "calculator"
}
func (c *Calculator) Handler(ctx context.Context, input string) (string, error) {
	v, err := starlark.Eval(&starlark.Thread{Name: "main"}, "input", input, math.Module.Members)
	if err != nil {
		return fmt.Sprintf("error from evaluator: %s", err.Error()), nil //nolint:nilerr
	}
	result := v.String()
	return result, nil
}

func (c *Calculator) Input() any {
	return nil
}

func (c *Calculator) Type() core.ToolType {
	return core.Normal
}
