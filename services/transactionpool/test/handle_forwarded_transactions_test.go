// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHandleForwardedTransactionsDiscardsMessagesWithInvalidSignature(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		ctrlRand := rand.NewControlledRand(t)
		h := newHarness(t).start(ctx)

		invalidSig := make([]byte, 32)
		ctrlRand.Read(invalidSig)

		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()

		_, err := h.txpool.HandleForwardedTransactions(ctx, &gossiptopics.ForwardedTransactionsInput{
			Message: &gossipmessages.ForwardedTransactionsMessage{
				Sender: (&gossipmessages.SenderSignatureBuilder{
					SenderNodeAddress: otherNodeKeyPair.NodeAddress(),
					Signature:         invalidSig,
				}).Build(),
				SignedTransactions: transactionpool.Transactions{tx1, tx2},
			},
		})

		require.Error(t, err, "did not fail on invalid signature")
	})
}

func TestHandleForwardedTransactionsAddsMessagesToPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).start(ctx)

		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()

		h.handleForwardFrom(ctx, otherNodeKeyPair, tx1, tx2)
		out, _ := h.getTransactionsForOrdering(ctx, 2, 2)
		require.Equal(t, 2, len(out.SignedTransactions), "forwarded transactions were not added to pool")
	})
}

func TestHandleForwardedTransactionsDoesNotAddToFullPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarnessWithSizeLimit(t, 1).allowingErrorsMatching("error adding forwarded transaction to pending pool").start(ctx)

		tx1 := builders.TransferTransaction().Build()

		h.handleForwardFrom(ctx, otherNodeKeyPair, tx1)
		out, _ := h.getTransactionsForOrdering(ctx, 2, 1)
		require.Equal(t, 0, len(out.SignedTransactions), "forwarded transaction was added to full pool")
	})
}
