// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const expirationWindowInterval = 30 * time.Minute
const futureTimestampGrace = 3 * time.Minute
const nodeSyncRejectInterval = 2 * time.Minute

func aValidationContextAsOf() *validationContext {
	return &validationContext{
		expiryWindow:           expirationWindowInterval,
		nodeSyncRejectInterval: nodeSyncRejectInterval,
		futureTimestampGrace:   futureTimestampGrace,
		virtualChainId:         builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
	}
}

func futureTimeAfterGracePeriod(lastCommittedBlockTimestamp primitives.TimestampNano) time.Time {
	return time.Unix(0, int64(lastCommittedBlockTimestamp)).Add(futureTimestampGrace + 1*time.Minute)
}

func TestValidateTransaction_Add_ValidTransaction(t *testing.T) {
	currentTime := time.Now()
	lastCommittedBlockTime := primitives.TimestampNano(currentTime.Add(nodeSyncRejectInterval / 2).UnixNano())
	err := aValidationContextAsOf().ValidateAddedTransaction(aTransactionAtNodeTimestamp(lastCommittedBlockTime).Build(), currentTime, lastCommittedBlockTime)
	require.Nil(t, err, "a valid transaction was rejected")
}

func TestValidateTransaction_Add_RejectsTransactionsWhenTimestampIsZero(t *testing.T) {
	vctx := &validationContext{
		expiryWindow:         expirationWindowInterval,
		futureTimestampGrace: futureTimestampGrace,
		virtualChainId:       builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
	}
	err := vctx.ValidateAddedTransaction(builders.TransferTransaction().Build(), time.Now(), 0)
	require.Error(t, err, "a transaction was not rejected when the system is in zero timestamp")
}

func TestValidateTransaction_Add_NodeOutOfSync(t *testing.T) {
	currentTime := time.Now()
	lastCommittedBlockTime := primitives.TimestampNano(currentTime.Add(-2 * nodeSyncRejectInterval).UnixNano())
	err := aValidationContextAsOf().ValidateAddedTransaction(builders.TransferTransaction().Build(), currentTime, lastCommittedBlockTime)
	require.Error(t, err, fmt.Sprintf("a transaction was not rejected when the node is out of sync"))
	require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_NODE_OUT_OF_SYNC, err.TransactionStatus, "error status differed from expected")
}

func TestValidateTransaction_Add_InvalidTransactions(t *testing.T) {
	currentTime := time.Now()
	lastCommittedBlockTime := primitives.TimestampNano(currentTime.Add(nodeSyncRejectInterval / 2).UnixNano())

	tests := []struct {
		name           string
		txBuilder      *builders.TransactionBuilder
		expectedStatus protocol.TransactionStatus
	}{
		{"protocol version", aTransactionAtNodeTimestamp(lastCommittedBlockTime).WithProtocolVersion(ProtocolVersion + 1), protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION},
		{"signer scheme", aTransactionAtNodeTimestamp(lastCommittedBlockTime).WithInvalidSignerScheme(), protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME},
		{"signer public key (wrong length)", aTransactionAtNodeTimestamp(lastCommittedBlockTime).WithInvalidPublicKey(), protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH},
		{"timestamp (created prior to the expiry window)", builders.TransferTransaction().WithTimestamp(currentTime.Add(expirationWindowInterval * -2)), protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED},
		{"timestamp (ahead of timestamp for last committed block)", builders.TransferTransaction().WithTimestamp(futureTimeAfterGracePeriod(lastCommittedBlockTime)), protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME},
		{"virtual chain id", aTransactionAtNodeTimestamp(lastCommittedBlockTime).WithVirtualChainId(primitives.VirtualChainId(1)), protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH},
	}
	for i := range tests {
		test := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(test.name, func(t *testing.T) {
			err := aValidationContextAsOf().ValidateAddedTransaction(test.txBuilder.Build(), currentTime, lastCommittedBlockTime)

			require.Error(t, err, fmt.Sprintf("a transaction with an invalid %s was not rejected", test.name))
			require.Equal(t, test.expectedStatus, err.TransactionStatus, "error status differed from expected")
		})
	}
}

func TestValidateTransaction_Ordering_InvalidTransactions(t *testing.T) {
	currentTime := time.Now()
	proposedBlockTime := primitives.TimestampNano(currentTime.Add(nodeSyncRejectInterval / 2).UnixNano())

	tests := []struct {
		name           string
		txBuilder      *builders.TransactionBuilder
		expectedStatus protocol.TransactionStatus
	}{
		{"protocol version", aTransactionAtNodeTimestamp(proposedBlockTime).WithProtocolVersion(ProtocolVersion + 1), protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION},
		{"signer scheme", aTransactionAtNodeTimestamp(proposedBlockTime).WithInvalidSignerScheme(), protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME},
		{"signer public key (wrong length)", aTransactionAtNodeTimestamp(proposedBlockTime).WithInvalidPublicKey(), protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH},
		{"timestamp (created prior to the expiry window)", builders.TransferTransaction().WithTimestamp(currentTime.Add(expirationWindowInterval * -2)), protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED},
		{"timestamp (ahead of timestamp for last committed block)", builders.TransferTransaction().WithTimestamp(futureTimeAfterGracePeriod(proposedBlockTime)), protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME},
		{"virtual chain id", aTransactionAtNodeTimestamp(proposedBlockTime).WithVirtualChainId(primitives.VirtualChainId(1)), protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH},
	}
	for i := range tests {
		test := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(test.name, func(t *testing.T) {
			err := aValidationContextAsOf().ValidateTransactionForOrdering(test.txBuilder.Build(), proposedBlockTime)

			require.Error(t, err, fmt.Sprintf("a transaction with an invalid %s was not rejected", test.name))
			require.Equal(t, test.expectedStatus, err.TransactionStatus, "error status differed from expected")
		})
	}
}

func aTransactionAtNodeTimestamp(lastCommittedBlockTimestamp primitives.TimestampNano) *builders.TransactionBuilder {
	return builders.TransferTransaction().WithTimestamp(time.Unix(0, int64(lastCommittedBlockTimestamp+1000)))
}
