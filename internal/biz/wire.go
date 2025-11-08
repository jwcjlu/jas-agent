package biz

import "github.com/google/wire"

// ProviderSet biz provider.
var ProviderSet = wire.NewSet(NewAgentUsecase)
