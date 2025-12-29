package service

import "github.com/google/wire"

// ProviderSet service provider.
var ProviderSet = wire.NewSet(NewAgentService, NewKnowledgeServiceImpl)
