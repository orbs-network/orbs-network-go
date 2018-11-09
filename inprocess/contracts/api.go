package contracts

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type ContractAPI interface {
	WaitForTransactionInState(ctx context.Context, txhash primitives.Sha256)
	SendTransaction(ctx context.Context, tx *protocol.SignedTransactionBuilder, nodeIndex int) chan *client.SendTransactionResponse
	SendTransactionInBackground(ctx context.Context, tx *protocol.SignedTransactionBuilder, nodeIndex int)
	CallMethod(ctx context.Context, tx *protocol.TransactionBuilder, nodeIndex int) chan uint64
}

type contractClient struct {
	API    ContractAPI
	logger log.BasicLogger
}

func NewContractClient(api ContractAPI, logger log.BasicLogger) *contractClient {
	return &contractClient{API: api, logger: logger}
}
