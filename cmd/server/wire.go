//go:build wireinject
// +build wireinject

package main

import (
	"errors"
	"jas-agent/agent/rag/embedding"

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
		newEmbedder,
	)
	return nil, nil, nil
}

func newChat(c *conf.Bootstrap) (llm.Chat, error) {
	if c.Llm == nil {
		return nil, errMissingLLMConfig
	}
	return llm.NewChat(&llm.Config{
		ApiKey:  c.Llm.ApiKey,
		BaseURL: c.Llm.BaseUrl,
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
func newEmbedder(c *conf.Bootstrap) embedding.Embedder {
	if c == nil {
		return nil
	}
	return embedding.NewOpenAIEmbedder(embedding.Config{
		ApiKey:  c.Llm.GetApiKey(),
		BaseURL: c.Llm.BaseUrl,
		Model:   "text-embedding-3-small",
	})
}
