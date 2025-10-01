package main

import (
	"flag"
	"fmt"
	"jas-agent/agent"
	"jas-agent/llm"

	"github.com/sashabaranov/go-openai"
	_ "jas-agent/examples/react/tools"
)

func main() {
	fmt.Println("Starting ReactAgent example...")
	var apiKey string
	var baseUrl string
	flag.StringVar(&apiKey, "apiKey", "apiKey", "apiKey")
	flag.StringVar(&baseUrl, "baseUrl", "baseUrl", "baseUrl")
	flag.Parse()
	chat := llm.NewChat(&llm.Config{
		ApiKey:  apiKey,
		BaseURL: baseUrl,
	})
	context := agent.NewContext(agent.WithModel(openai.GPT3Dot5Turbo), agent.WithChat(chat))
	executor := agent.NewAgentExecutor(context)
	fmt.Println("Running agent with query...")
	result := executor.Run("我有3只狗,分别是border collie ,scottish terrier和toy poodle.请问这些的体重总和")

	fmt.Printf("Final result: %s\n", result)
}
