// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
	GetTransactionReceiptProof(ctx context.Context, txHash primitives.Sha256, nodeIndex int) *client.GetTransactionReceiptProofResponse
	GetVirtualChainId() primitives.VirtualChainId
}

type contractClient struct {
	API CallContractAPI
}

func NewContractClient(api CallContractAPI) *contractClient {
	return &contractClient{API: api}
}
