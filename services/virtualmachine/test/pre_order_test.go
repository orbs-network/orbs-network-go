package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_GlobalPreOrder"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPreOrderDifferentSignerSchemes(t *testing.T) {
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness()

			h.expectSystemContractCalled(globalpreorder.CONTRACT.Name, globalpreorder.METHOD_APPROVE.Name, nil)

			results, err := h.transactionSetPreOrder([]*protocol.SignedTransaction{test.tx})
			if test.status == protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
				require.NoError(t, err, "transaction set pre order should not fail")
			} else {
				require.Error(t, err, "transaction set pre order should fail")
			}
			require.Equal(t, []protocol.TransactionStatus{test.status}, results, "transactionSetPreOrder returned statuses should match")

			h.verifySystemContractCalled(t)
		})
	}
}

func TestPreOrderGlobalSubscriptionContractNotApproved(t *testing.T) {
	h := newHarness()

	h.expectSystemContractCalled(globalpreorder.CONTRACT.Name, globalpreorder.METHOD_APPROVE.Name, errors.New("contract not approved"))

	tx := builders.Transaction().Build()
	results, err := h.transactionSetPreOrder([]*protocol.SignedTransaction{tx})
	require.Error(t, err, "transaction set pre order should fail")
	require.Equal(t, []protocol.TransactionStatus{protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER}, results, "transactionSetPreOrder returned statuses should match")

	h.verifySystemContractCalled(t)
}
