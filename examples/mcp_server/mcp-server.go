package main

import (
	"fmt"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"

	"github.com/Knetic/govaluate"
)

// Calculate 工具的参数
type CalculateArgs struct {
	Expression string `json:"expression" jsonschema:"required,description=要计算的表达式,示例: 4 * 7 / 3"`
}

// AverageDogWeight 工具的参数
type AverageDogWeightArgs struct {
	Name string `json:"name" jsonschema:"required,description=狗的品种名称,示例: Border Collie"`
}

// RegisterMCPTools 在给定的 MCP 服务器上注册工具
func RegisterMCPTools(server *mcp_golang.Server) error {
	// 注册 calculate
	if err := server.RegisterTool("calculate", "执行表达式计算，示例: 4 * 7 / 3", func(args CalculateArgs) (*mcp_golang.ToolResponse, error) {
		// 优先直接计算表达式，避免复用现有 Calculate 的空字符串返回
		expr, err := govaluate.NewEvaluableExpression(args.Expression)
		if err != nil {
			return nil, fmt.Errorf("表达式错误: %v", err)
		}
		res, err := expr.Evaluate(nil)
		if err != nil {
			return nil, fmt.Errorf("计算失败: %v", err)
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("%v", res))), nil
	}); err != nil {
		return err
	}

	// 注册 average_dog_weight
	if err := server.RegisterTool("average_dog_weight", "根据狗的品种返回平均体重", func(args AverageDogWeightArgs) (*mcp_golang.ToolResponse, error) {
		result := AverageDogWeight(args.Name)
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(result)), nil
	}); err != nil {
		return err
	}

	return nil
}

// StartMCPHTTPServer 启动基于 HTTP 的 MCP 服务，监听端口 8099
func main() {
	transport := http.NewHTTPTransport("/mcp")
	transport.WithAddr(":8099")

	server := mcp_golang.NewServer(transport)
	if err := RegisterMCPTools(server); err != nil {
		panic(err)
	}
	if err := server.Serve(); err != nil {
		panic(err)
	}
}
func AverageDogWeight(name string) string {
	name = strings.ToLower(name)
	if name == strings.ToLower("Scottish Terrier") {
		return "Scottish Terriers average 20 lbs"
	} else if name == strings.ToLower("Border Collie") {
		return "a Border Collies average weight is 37 lbs"
	} else if name == strings.ToLower("Toy Poodle") {
		return "a toy poodles average weight is 7 lbs"
	} else {
		return "An average dog weights 50 lbs"
	}
}
