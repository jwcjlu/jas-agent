package main

import (
	"flag"
	"os"

	"github.com/sashabaranov/go-openai"

	"github.com/go-kratos/kratos/v2/log"
	agent "jas-agent/agent/agent"
	_ "jas-agent/agent/examples/react/tools"
	"jas-agent/agent/llm"
)

var logger = log.NewHelper(log.With(log.NewStdLogger(os.Stdout), "module", "examples/react"))

func main() {
	logger.Info("Starting ReactAgent example...")
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
	logger.Info("Running agent with query...")
	result := executor.Run("我有3只狗,分别是border collie ,scottish terrier和toy poodle.请问这些的体重总和")

	logger.Infof("Final result: %s", result)
}
