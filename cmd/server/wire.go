//go:build wireinject
// +build wireinject

package main

import (
	"errors"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"jas-agent/agent/llm"
	"jas-agent/internal/biz"
	"jas-agent/internal/conf"
	"jas-agent/internal/data"
	"jas-agent/internal/server"
	"jas-agent/internal/service"
)

var errMissingLLMConfig = errors.New("llm config is required")

func wireApp(c *conf.Bootstrap, logger log.Logger) (*kratos.App, func(), error) {
	wire.Build(
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		server.ProviderSet,
		newChat,
		provideServerConfig,
		provideDataConfig,
		provideLLMConfig,
	)
	return nil, nil, nil
}

func newChat(c *conf.LLM) (llm.Chat, error) {
	if c == nil {
		return nil, errMissingLLMConfig
	}
	return llm.NewChat(&llm.Config{
		ApiKey:  c.APIKey,
		BaseURL: c.BaseURL,
	}), nil
}

func provideServerConfig(c *conf.Bootstrap) *conf.Server {
	if c == nil {
		return nil
	}
	return c.Server
}

func provideDataConfig(c *conf.Bootstrap) *conf.Data {
	if c == nil {
		return nil
	}
	return c.Data
}

func provideLLMConfig(c *conf.Bootstrap) *conf.LLM {
	if c == nil {
		return nil
	}
	return c.LLM
}
