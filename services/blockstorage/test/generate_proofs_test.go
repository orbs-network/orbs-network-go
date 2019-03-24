// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGenerateProof(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).withSyncBroadcast(1).start(ctx)

		blockCreated := time.Now()
		blockHeight := primitives.BlockHeight(1)
		rxHash := hash.CalcSha256([]byte("just text"))
		block := builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).WithTransactions(3).
			WithReceiptsForTransactions().WithReceiptProofHash(rxHash).Build()
		_, err := harness.commitBlock(ctx, block)
		require.NoError(t, err)

		txHash := digest.CalcTxHash(block.TransactionsBlock.SignedTransactions[2].Transaction())

		proof, err := harness.blockStorage.GenerateReceiptProof(ctx, &services.GenerateReceiptProofInput{
			Txhash:      txHash,
			BlockHeight: blockHeight,
		})

		// very specific case where proof is actually only one sha
		require.NoError(t, err)
		merkleProof := proof.Proof.ReceiptProof()
		require.Len(t, merkleProof, 32, "should be one sha long")
		// calc the specific proof - is sha of receipt of 0, 1
		altProof := hashTwoRecpeits(block.ResultsBlock.TransactionReceipts, 0, 1)

		require.EqualValues(t, altProof, merkleProof)
		require.EqualValues(t, blockHeight, proof.Proof.Header().BlockHeight(), "wrong height")
		require.EqualValues(t, rxHash, proof.Proof.BlockProof().TransactionsBlockHash(), "wrong height")
	})
}

func TestGenerateProof_WrongTxHash(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).withSyncBroadcast(1).start(ctx)

		blockCreated := time.Now()
		blockHeight := primitives.BlockHeight(1)
		block := builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).WithTransactions(3).Build()
		_, err := harness.commitBlock(ctx, block)
		require.NoError(t, err)

		fakeTxHash := hash.CalcSha256([]byte("any text"))

		proof, err := harness.blockStorage.GenerateReceiptProof(ctx, &services.GenerateReceiptProofInput{
			Txhash:      fakeTxHash,
			BlockHeight: blockHeight,
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "could not find transaction inside block", "expected not found err")
		require.Nil(t, proof, "proof should have been nil")

	})
}

func hashTwoRecpeits(list []*protocol.TransactionReceipt, index0, index1 int) primitives.Sha256 {
	l0 := digest.CalcReceiptHash(list[index0])
	l1 := digest.CalcReceiptHash(list[index1])
	if bytes.Compare(l0, l1) > 0 {
		return hash.CalcSha256(l1, l0)
	}
	return hash.CalcSha256(l0, l1)
}
