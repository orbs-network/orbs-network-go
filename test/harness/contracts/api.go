package contracts

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type APIProvider interface {
	GetPublicApi() services.PublicApi
	GetCompiler() adapter.Compiler
}
