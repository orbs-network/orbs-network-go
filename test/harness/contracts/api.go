package contracts

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

//TODO abstract public API's methods
//TODO mocking the compiler shouldn't be here
type APIProvider interface {
	GetPublicApi() services.PublicApi
	GetCompiler() adapter.Compiler
	WaitForTransactionInStateForAtMost(ctx context.Context, txhash primitives.Sha256, atMost time.Duration)
}

type contractClient struct {
	apis   []APIProvider
	logger log.BasicLogger
}

func NewContractClient(apis []APIProvider, logger log.BasicLogger) *contractClient {
	return &contractClient{apis: apis, logger:logger}
}