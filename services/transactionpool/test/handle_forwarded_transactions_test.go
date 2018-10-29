package test

import (
	"context"
	"crypto/rand"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHandleForwardedTransactionsDiscardsMessagesWithInvalidSignature(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()

		invalidSig := make([]byte, 32)
		rand.Read(invalidSig)

		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()

		_, err := h.txpool.HandleForwardedTransactions(ctx, &gossiptopics.ForwardedTransactionsInput{
			Message: &gossipmessages.ForwardedTransactionsMessage{
				Sender: (&gossipmessages.SenderSignatureBuilder{
					SenderPublicKey: otherNodeKeyPair.PublicKey(),
					Signature:       invalidSig,
				}).Build(),
				SignedTransactions: transactionpool.Transactions{tx1, tx2},
			},
		})

		require.Error(t, err, "did not fail on invalid signature")
	})
}

func TestHandleForwardedTransactionsAddsMessagesToPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()

		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()

		h.handleForwardFrom(ctx, otherNodeKeyPair, tx1, tx2)
		out, _ := h.getTransactionsForOrdering(ctx, 2)
		require.Equal(t, 2, len(out.SignedTransactions), "forwarded transactions were not added to pool")
	})
}

func TestHandleForwardedTransactionsDoesNotAddToFullPool(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarnessWithSizeLimit(1)

		tx1 := builders.TransferTransaction().Build()

		h.handleForwardFrom(ctx, otherNodeKeyPair, tx1)
		out, _ := h.getTransactionsForOrdering(ctx, 1)
		require.Equal(t, 0, len(out.SignedTransactions), "forwarded transaction was added to full pool")
	})
}
