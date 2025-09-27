package main

import (
	"context"
	"jas-agent/tools"
)

type AverageDogWeight struct {
}

func init() {
	tools.GetToolManager().RegisterTool(AverageDogWeight{})
}
func (adw AverageDogWeight) Handler(ctx context.Context, name string) (string, error) {
	if name == "Scottish Terrier" {
		return "Scottish Terriers average 20 lbs", nil
	} else if name == "Border Collie" {
		return "a Border Collies average weight is 37 lbs", nil
	} else if name == "Toy Poodle" {
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
