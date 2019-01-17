package callcontract

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type CallContractAPI interface {
	SendTransaction(ctx context.Context, builder *protocol.SignedTransactionBuilder, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256)
	SendTransactionInBackground(ctx context.Context, builder *protocol.SignedTransactionBuilder, nodeIndex int)
	RunQuery(ctx context.Context, builder *protocol.SignedQueryBuilder, nodeIndex int) *client.RunQueryResponse
	GetTransactionStatus(ctx context.Context, txHash primitives.Sha256, nodeIndex int) *client.GetTransactionStatusResponse
}

type contractClient struct {
	API CallContractAPI
}

func NewContractClient(api CallContractAPI) *contractClient {
	return &contractClient{API: api}
}
