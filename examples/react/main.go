package main

import (
	"flag"
	"fmt"
	"jas-agent/agent"
	"jas-agent/llm"

	"github.com/sashabaranov/go-openai"
)

func main() {
	fmt.Println("Starting ReactAgent example...")
	var apiKey string
	var baseUrl string
	flag.StringVar(&apiKey, "apiKey", "apiKey", "apiKey")
	flag.StringVar(&baseUrl, "baseUrl", "baseUrl", "baseUrl")
	chat := llm.NewChat(&llm.Config{
		ApiKey:  apiKey,
		BaseURL: baseUrl,
	})

	context := agent.NewContext(agent.WithModel(openai.GPT3Dot5Turbo), agent.WithChat(chat))
	executor := agent.NewAgentExecutor(context)

	fmt.Println("Running agent with query...")
	result := executor.Run("I have 3 dogs, a border collie , a scottish terrier and a toy poodle. What is their combined weight")

	fmt.Printf("Final result: %s\n", result)
}
