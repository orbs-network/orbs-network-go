package contracts

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type CounterClient interface {
	SendDeployCounterContract(ctx context.Context, nodeIndex int) chan *client.SendTransactionResponse
	SendCounterAdd(ctx context.Context, nodeIndex int, amount uint64) chan *client.SendTransactionResponse
	CallCounterGet(ctx context.Context, nodeIndex int) chan uint64
}

func (c *contractClient) SendDeployCounterContract(ctx context.Context, nodeIndex int) chan *client.SendTransactionResponse {
	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	tx := builders.Transaction().
		WithMethod("_Deployments", "deployService").
		WithArgs(
			fmt.Sprintf("CounterFrom%d", counterStart),
			uint32(protocol.PROCESSOR_TYPE_NATIVE),
			[]byte(contracts.NativeSourceCodeForCounter(counterStart)),
		).Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}

func (c *contractClient) SendCounterAdd(ctx context.Context, nodeIndex int, amount uint64) chan *client.SendTransactionResponse {
	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	tx := builders.Transaction().
		WithMethod(primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)), "add").
		WithArgs(amount).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}

func (c *contractClient) CallCounterGet(ctx context.Context, nodeIndex int) chan uint64 {
	ch := make(chan uint64, 1)

	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	tx := builders.NonSignedTransaction().
		WithMethod(primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)), "get").
		Builder()

	select {
	case r := <-c.API.CallMethod(ctx, tx, nodeIndex):
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(r)
		ch <- outputArgsIterator.NextArguments().Uint64Value()
	case <-ctx.Done():
	}

	return ch
}
