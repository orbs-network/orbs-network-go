// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package callcontract

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type ElectionsClient interface {
	UnsafeTests_SetElectedValidators(ctx context.Context, nodeIndex int, electedValidatorIndexes []int) (*client.SendTransactionResponse, primitives.Sha256)
}

func (c *contractClient) UnsafeTests_SetElectedValidators(ctx context.Context, nodeIndex int, electedValidatorIndexes []int) (*client.SendTransactionResponse, primitives.Sha256) {
	joinedElectedValidatorAddresses := []byte{}
	for _, electedValidatorIndex := range electedValidatorIndexes {
		address := testKeys.EcdsaSecp256K1KeyPairForTests(electedValidatorIndex).NodeAddress()
		joinedElectedValidatorAddresses = append(joinedElectedValidatorAddresses, address...)
	}
	if len(joinedElectedValidatorAddresses) != digest.NODE_ADDRESS_SIZE_BYTES*len(electedValidatorIndexes) {
		panic("joinedElectedValidatorAddresses length is invalid")
	}

	tx := builders.Transaction().
		WithVirtualChainId(c.API.GetVirtualChainId()).
		WithMethod("_Elections", "unsafetests_setElectedValidators").
		WithArgs(joinedElectedValidatorAddresses).
		WithEd25519Signer(keys.Ed25519KeyPairForTests(0)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}
