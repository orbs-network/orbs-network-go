package test

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateTransactionsForOrderingAcceptsOkTransactions(t *testing.T) {
	t.Parallel()
	h := newHarness()

	require.NoError(t,
		h.validateTransactionsForOrdering(0, builders.Transaction().Build(), builders.Transaction().Build()),
		"rejected a set of valid transactions")
}

func TestValidateTransactionsForOrderingRejectsCommittedTransactions(t *testing.T) {
	t.Parallel()
	h := newHarness()

	h.ignoringForwardMessages()
	h.ignoringTransactionResults()

	committedTx := builders.Transaction().Build()

	h.addNewTransaction(committedTx)
	h.assumeBlockStorageAtHeight(1)
	h.reportTransactionsAsCommitted(committedTx)

	require.EqualErrorf(t,
		h.validateTransactionsForOrdering(0, committedTx, builders.Transaction().Build()),
		fmt.Sprintf("transaction with hash %s already committed", digest.CalcTxHash(committedTx.Transaction())),
		"did not reject a committed transaction")

}

func TestValidateTransactionsForOrderingRejectsTransactionsFailingValidation(t *testing.T) {
	t.Parallel()
	h := newHarness()

	invalidTx := builders.TransferTransaction().WithInvalidTimestamp().Build()

	require.EqualErrorf(t,
		h.validateTransactionsForOrdering(0, builders.Transaction().Build(), invalidTx),
		fmt.Sprintf("transaction with hash %s is invalid: transaction rejected: TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED", digest.CalcTxHash(invalidTx.Transaction())),
		"did not reject an invalid transaction")
}

func TestValidateTransactionsForOrderingRejectsTransactionsFailingPreOrderChecks(t *testing.T) {
	t.Parallel()
	h := newHarness()

	invalidTx := builders.TransferTransaction().Build()
	h.failPreOrderCheckFor(func(tx *protocol.SignedTransaction) bool {
		return tx == invalidTx
	})

	require.EqualErrorf(t,
		h.validateTransactionsForOrdering(0, builders.Transaction().Build(), invalidTx),
		fmt.Sprintf("transaction with hash %s failed pre-order checks with status TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER", digest.CalcTxHash(invalidTx.Transaction())),
		"did not reject transaction that failed pre-order checks")

}

func TestValidateTransactionsForOrderingRejectsBlockHeightOutsideOfGrace(t *testing.T) {
	t.Parallel()
	h := newHarness()

	require.EqualErrorf(t,
		h.validateTransactionsForOrdering(666, builders.Transaction().Build()),
		"requested future block outside of grace range",
		"did not reject block height too far in the future")

}
