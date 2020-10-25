// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateTransactionsForOrderingAcceptsOkTransactions(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		require.NoError(t,
			h.validateTransactionsForOrdering(ctx, 2, config.MAXIMAL_CONSENSUS_BLOCK_PROTOCOL_VERSION, builders.Transaction().Build(), builders.Transaction().Build()),
			"rejected a set of valid transactions")
	})
}

func TestValidateTransactionsForOrderingRejectsCommittedTransactions(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		h.ignoringForwardMessages()
		h.ignoringTransactionResults()

		committedTx := builders.Transaction().Build()

		h.addNewTransaction(ctx, committedTx)
		h.assumeBlockStorageAtHeight(1)
		h.reportTransactionsAsCommitted(ctx, committedTx)

		require.EqualErrorf(t,
			h.validateTransactionsForOrdering(ctx, 2, config.MAXIMAL_CONSENSUS_BLOCK_PROTOCOL_VERSION, committedTx, builders.Transaction().Build()),
			fmt.Sprintf("transaction with hash %s already committed", digest.CalcTxHash(committedTx.Transaction())),
			"did not reject a committed transaction")
	})
}

func TestValidateTransactionsForOrderingRejectsTransactionsFailingValidation(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		invalidTx := builders.TransferTransaction().WithTimestampInFarFuture().Build()

		err := h.validateTransactionsForOrdering(ctx, 1, invalidTx.Transaction().ProtocolVersion(), builders.Transaction().Build(), invalidTx)

		require.Contains(t,
			err.Error(),
			fmt.Sprintf("transaction with hash %s is invalid: transaction rejected: %s", digest.CalcTxHash(invalidTx.Transaction()), protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME),
			"did not reject an invalid transaction")
	})
}

func TestValidateTransactionsForOrderingRejectsTransactionsFailingPreOrderChecks(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		invalidTx := builders.TransferTransaction().Build()
		h.failPreOrderCheckFor(func(tx *protocol.SignedTransaction) bool {
			return tx == invalidTx
		}, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER)

		require.EqualErrorf(t,
			h.validateTransactionsForOrdering(ctx, 2, config.MAXIMAL_CONSENSUS_BLOCK_PROTOCOL_VERSION, builders.Transaction().Build(), invalidTx),
			fmt.Sprintf("transaction with hash %s failed pre-order checks with status TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER", digest.CalcTxHash(invalidTx.Transaction())),
			"did not reject transaction that failed pre-order checks")
	})
}

func TestValidateTransactionsForOrderingRejectsBlockHeightOutsideOfGrace(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		require.EqualErrorf(t,
			h.validateTransactionsForOrdering(ctx, 666, config.MAXIMAL_CONSENSUS_BLOCK_PROTOCOL_VERSION, builders.Transaction().Build()),
			"requested future block outside of grace range",
			"did not reject block height too far in the future")
	})
}

func TestValidateTransactionsForOrderingRejectsClientProtocolVersionLargerThanMaxiumu(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newHarness(parent).start(ctx)

		err := h.validateTransactionsForOrdering(ctx, 2, config.MAXIMAL_CONSENSUS_BLOCK_PROTOCOL_VERSION, builders.Transaction().WithProtocolVersion(config.MAXIMAL_CLIENT_PROTOCOL_VERSION+1).Build())
		require.Contains(t, err.Error(),
			fmt.Sprintf("transaction rejected: TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION (expected %d but got %d)", config.MAXIMAL_CLIENT_PROTOCOL_VERSION, config.MAXIMAL_CLIENT_PROTOCOL_VERSION+1),
			"did not reject tx client protocol version larger than max")
	})
}
