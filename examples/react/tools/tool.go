package tools

import (
	"context"
	"jas-agent/core"
	"jas-agent/tools"
	"strings"
)

type AverageDogWeight struct {
}

func init() {
	tools.GetToolManager().RegisterTool(&AverageDogWeight{})
}
func (adw AverageDogWeight) Handler(ctx context.Context, name string) (string, error) {
	if strings.ToLower(name) == strings.ToLower("Scottish Terrier") {
		return "Scottish Terriers average 20 lbs", nil
	} else if strings.ToLower(name) == strings.ToLower("Border Collie") {
		return "a Border Collies average weight is 37 lbs", nil
	} else if strings.ToLower(name) == strings.ToLower("Toy Poodle") {
		return "a toy poodles average weight is 7 lbs", nil
	} else {
		return "An average dog weights 50 lbs", nil
	}
}

// Description returns a string describing the calculator tool.
func (adw AverageDogWeight) Description() string {
	return `e.g. averageDogWeight: Collie
       returns average weight of a dog when given the breed`
}

// Name returns the name of the tool.
func (adw AverageDogWeight) Name() string {
	return "averageDogWeight"
}

func (adw AverageDogWeight) Input() any {
	return nil
}
func (adw AverageDogWeight) Type() core.ToolType {
	return core.Normal
}
