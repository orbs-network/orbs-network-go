package contracts

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type ContractAPI interface {
	WaitForTransactionInState(ctx context.Context, txhash primitives.Sha256)
	SendTransaction(ctx context.Context, tx *protocol.SignedTransactionBuilder, nodeIndex int) *client.SendTransactionResponse
	SendTransactionInBackground(ctx context.Context, tx *protocol.SignedTransactionBuilder, nodeIndex int)
	CallMethod(ctx context.Context, tx *protocol.TransactionBuilder, nodeIndex int) *client.CallMethodResponse
}

type contractClient struct {
	API ContractAPI
}

func NewContractClient(api ContractAPI) *contractClient {
	return &contractClient{API: api}
}
