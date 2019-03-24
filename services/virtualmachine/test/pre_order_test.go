// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/GlobalPreOrder"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
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
			test.WithContext(func(ctx context.Context) {
				h := newHarness(t)

				h.expectSystemContractCalled(globalpreorder_systemcontract.CONTRACT_NAME, globalpreorder_systemcontract.METHOD_APPROVE, nil)

				results, err := h.transactionSetPreOrder(ctx, []*protocol.SignedTransaction{tt.tx})
				require.NoError(t, err, "transaction set pre order should not fail on signature problems")
				require.Equal(t, []protocol.TransactionStatus{tt.status}, results, "transactionSetPreOrder returned statuses should match")

				h.verifySystemContractCalled(t)
			})
		})
	}
}

func TestPreOrder_GlobalSubscriptionContractNotApproved(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		h.expectSystemContractCalled(globalpreorder_systemcontract.CONTRACT_NAME, globalpreorder_systemcontract.METHOD_APPROVE, errors.New("subscription problem"))

		txs := []*protocol.SignedTransaction{}
		txs = append(txs, builders.TransferTransaction().Build())
		txs = append(txs, builders.Transaction().WithContract(globalpreorder_systemcontract.CONTRACT_NAME).Build())
		txs = append(txs, builders.TransferTransaction().Build())

		results, err := h.transactionSetPreOrder(ctx, txs)
		require.NoError(t, err, "transaction set pre order should not fail")
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, results[0], "first tx should be rejected")
		require.Equal(t, protocol.TRANSACTION_STATUS_PRE_ORDER_VALID, results[1], "second tx should not be rejected since it is made to _GlobalPreOrder")
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, results[2], "third tx should be rejected")

		h.verifySystemContractCalled(t)
	})
}
