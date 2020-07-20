// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPreOrder_DifferentSignerSchemes(t *testing.T) {
	tests := []struct {
		name   string
		tx     *protocol.SignedTransaction
		status protocol.TransactionStatus
	}{
		{
			name:   "UnknownSignerScheme",
			tx:     builders.Transaction().WithInvalidSignerScheme().Build(),
			status: protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME,
		},
		{
			name:   "InvalidEd25519Signature",
			tx:     builders.Transaction().WithInvalidEd25519Signer(keys.Ed25519KeyPairForTests(1)).Build(),
			status: protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH,
		},
		{
			name:   "ValidEd25519Signature",
			tx:     builders.Transaction().WithEd25519Signer(keys.Ed25519KeyPairForTests(1)).Build(),
			status: protocol.TRANSACTION_STATUS_PRE_ORDER_VALID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			with.Context(func(ctx context.Context) {
				with.Logging(t, func(parent *with.LoggingHarness) {
					h := newHarness(parent.Logger)

					results, err := h.transactionSetPreOrder(ctx, []*protocol.SignedTransaction{tt.tx})
					require.NoError(t, err, "transaction set pre order should not fail on signature problems")
					require.Equal(t, []protocol.TransactionStatus{tt.status}, results, "transactionSetPreOrder returned statuses should match")

					h.verifySystemContractCalled(t)
				})
			})
		})
	}
}

func TestPreOrder_SubscriptionNotApproved(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger)
			h.management.Reset()
			h.management.When("GetSubscriptionStatus", mock.Any, mock.Any).Return(&services.GetSubscriptionStatusOutput{SubscriptionStatusIsActive: false}, nil)

			txs := []*protocol.SignedTransaction{}
			txs = append(txs, builders.TransferTransaction().Build())
			txs = append(txs, builders.TransferTransaction().Build())

			results, err := h.transactionSetPreOrder(ctx, txs)
			require.NoError(t, err, "transaction set pre order should not fail")
			require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, results[0], "first tx should be rejected")
			require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, results[1], "third tx should be rejected")
		})
	})
}

func TestPreOrder_NetworkTimeReferenceTooOld(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			parent.AllowErrorsMatching("Network has lost live connection to management")
			h := newHarness(parent.Logger)

			txs := []*protocol.SignedTransaction{}
			txs = append(txs, builders.TransferTransaction().Build())
			txs = append(txs, builders.TransferTransaction().Build())

			output, err := h.service.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
				SignedTransactions:        txs,
				CurrentBlockHeight:        12,
				CurrentBlockTimestamp:     primitives.TimestampNano(time.Now().UnixNano()),
				CurrentBlockReferenceTime: primitives.TimestampSeconds(time.Now().Add(-h.cfg.CommitteeValidityTimeout()).Add(-1 * time.Hour).Unix()),
			})

			require.NoError(t, err, "transaction set pre order should fail")
			require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, output.PreOrderResults[0], "first tx should be rejected")
			require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, output.PreOrderResults[1], "third tx should be rejected")
		})
	})
}
