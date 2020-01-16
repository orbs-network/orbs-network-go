// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package callcontract

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type CommitteeClient interface {
	GetOrderedCommittee(ctx context.Context, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256)
	GetAllCommitteeMisses(ctx context.Context, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256)
}

func (c *contractClient) GetOrderedCommittee(ctx context.Context, nodeIndex int) *client.RunQueryResponse {
	tx := builders.Query().
		WithVirtualChainId(c.API.GetVirtualChainId()).
		WithMethod("_Committee", "getOrderedCommittee").
		WithArgs().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(0)).
		Builder()

	return c.API.RunQuery(ctx, tx, nodeIndex)
}

func (c *contractClient) GetAllCommitteeMisses(ctx context.Context, nodeIndex int) *client.RunQueryResponse {
	tx := builders.Query().
		WithVirtualChainId(c.API.GetVirtualChainId()).
		WithMethod("_Committee", "getAllCommitteeMisses").
		WithArgs().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(0)).
		Builder()

	return c.API.RunQuery(ctx, tx, nodeIndex)
}
