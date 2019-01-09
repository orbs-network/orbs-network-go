package callcontract

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type CallContractAPI interface {
	WaitForTransactionInState(ctx context.Context, txHash primitives.Sha256)
	SendTransaction(ctx context.Context, builder *protocol.SignedTransactionBuilder, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256)
	SendTransactionInBackground(ctx context.Context, builder *protocol.SignedTransactionBuilder, nodeIndex int)
	RunQuery(ctx context.Context, builder *protocol.SignedQueryBuilder, nodeIndex int) *client.RunQueryResponse
}

type contractClient struct {
	API CallContractAPI
}

func NewContractClient(api CallContractAPI) *contractClient {
	return &contractClient{API: api}
}
