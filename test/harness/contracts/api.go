package contracts

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

//TODO abstract public API's methods
//TODO mocking the compiler shouldn't be here
type APIProvider interface {
	GetPublicApi() services.PublicApi
	GetCompiler() adapter.Compiler
	WaitForTransactionInStateForAtMost(ctx context.Context, txhash primitives.Sha256, atMost time.Duration) // TODO remove atMost and use context with timeout
}

type contractClient struct {
	apis   []APIProvider
	logger log.BasicLogger
}

func NewContractClient(apis []APIProvider, logger log.BasicLogger) *contractClient {
	return &contractClient{apis: apis, logger: logger}
}

func (c *contractClient) sendTransaction(ctx context.Context, tx *protocol.SignedTransactionBuilder, nodeIndex int) chan *client.SendTransactionResponse {
	ch := make(chan *client.SendTransactionResponse)
	supervised.GoOnce(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{SignedTransaction: tx}).Build(),
		})
		if err != nil {
			panic(fmt.Sprintf("error sending transaction: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse
	})
	return ch
}

func (c *contractClient) callMethod(ctx context.Context, tx *protocol.TransactionBuilder, nodeIndex int) chan uint64 {

	ch := make(chan uint64)
	supervised.GoOnce(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		output, err := publicApi.CallMethod(ctx, &services.CallMethodInput{
			ClientRequest: (&client.CallMethodRequestBuilder{Transaction: tx}).Build(),
		})
		if err != nil {
			panic(fmt.Sprintf("error calling method: %v", err)) // TODO: improve
		}
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(output.ClientResponse)
		ch <- outputArgsIterator.NextArguments().Uint64Value()
	})
	return ch
}
