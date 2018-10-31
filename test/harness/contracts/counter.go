package contracts

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/synchronization/supervized"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type CounterClient interface {
	SendDeployCounterContract(ctx context.Context, nodeIndex int) chan *client.SendTransactionResponse
	SendCounterAdd(ctx context.Context, nodeIndex int, amount uint64) chan *client.SendTransactionResponse
	CallCounterGet(ctx context.Context, nodeIndex int) chan uint64
}

func (c *contractClient) SendDeployCounterContract(ctx context.Context, nodeIndex int) chan *client.SendTransactionResponse {
	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	// if needed, provide a fake implementation of this contract to all nodes
	for _, api := range c.apis {
		if fakeCompiler, ok := api.GetCompiler().(adapter.FakeCompiler); ok {
			fakeCompiler.ProvideFakeContract(contracts.MockForCounter(), string(contracts.NativeSourceCodeForCounter(counterStart)))
		}
	}

	tx := builders.Transaction().
		WithMethod("_Deployments", "deployService").
		WithArgs(
			fmt.Sprintf("CounterFrom%d", counterStart),
			uint32(protocol.PROCESSOR_TYPE_NATIVE),
			[]byte(contracts.NativeSourceCodeForCounter(counterStart)),
		)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: tx.Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	supervized.ShortLived(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error sending counter deploy: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse
	})
	return ch
}

func (c *contractClient) SendCounterAdd(ctx context.Context, nodeIndex int, amount uint64) chan *client.SendTransactionResponse {
	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	tx := builders.Transaction().
		WithMethod(primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)), "add").
		WithArgs(amount)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: tx.Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	supervized.ShortLived(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error sending counter add for the amount %d: %v", amount, err)) // TODO: improve
		}
		ch <- output.ClientResponse
	})
	return ch
}

func (c *contractClient) CallCounterGet(ctx context.Context, nodeIndex int) chan uint64 {
	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	request := (&client.CallMethodRequestBuilder{
		Transaction: builders.NonSignedTransaction().
			WithMethod(primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)), "get").Builder(),
	}).Build()

	ch := make(chan uint64)
	supervized.ShortLived(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		output, err := publicApi.CallMethod(ctx, &services.CallMethodInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in calling counter get: %v", err)) // TODO: improve
		}
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(output.ClientResponse)
		ch <- outputArgsIterator.NextArguments().Uint64Value()
	})
	return ch
}