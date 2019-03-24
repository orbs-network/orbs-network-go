// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestResults_TranslateTransactionStatusToRequestStatus(t *testing.T) {
	tests := []struct {
		name     string
		expect   protocol.RequestStatus
		txStatus protocol.TransactionStatus
		exResult protocol.ExecutionResult
	}{
		{"TRANSACTION_STATUS_RESERVED", protocol.REQUEST_STATUS_RESERVED, protocol.TRANSACTION_STATUS_RESERVED, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_COMMITTED+EXECUTION_RESULT_SUCCESS", protocol.REQUEST_STATUS_COMPLETED, protocol.TRANSACTION_STATUS_COMMITTED, protocol.EXECUTION_RESULT_SUCCESS},
		{"TRANSACTION_STATUS_COMMITTED+EXECUTION_RESULT_ERROR_SMART_CONTRACT", protocol.REQUEST_STATUS_COMPLETED, protocol.TRANSACTION_STATUS_COMMITTED, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT},
		{"TRANSACTION_STATUS_COMMITTED+EXECUTION_RESULT_ERROR_INPUT", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_COMMITTED, protocol.EXECUTION_RESULT_ERROR_INPUT},
		{"TRANSACTION_STATUS_COMMITTED+EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_COMMITTED, protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED},
		{"TRANSACTION_STATUS_COMMITTED+EXECUTION_RESULT_ERROR_UNEXPECTED", protocol.REQUEST_STATUS_SYSTEM_ERROR, protocol.TRANSACTION_STATUS_COMMITTED, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED},
		{"TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED+EXECUTION_RESULT_SUCCESS", protocol.REQUEST_STATUS_COMPLETED, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, protocol.EXECUTION_RESULT_SUCCESS},
		{"TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED+EXECUTION_RESULT_ERROR_SMART_CONTRACT", protocol.REQUEST_STATUS_COMPLETED, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT},
		{"TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED+EXECUTION_RESULT_ERROR_INPUT", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, protocol.EXECUTION_RESULT_ERROR_INPUT},
		{"TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED+EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED},
		{"TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED+EXECUTION_RESULT_ERROR_UNEXPECTED", protocol.REQUEST_STATUS_SYSTEM_ERROR, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED},
		{"TRANSACTION_STATUS_PENDING", protocol.REQUEST_STATUS_IN_PROCESS, protocol.TRANSACTION_STATUS_PENDING, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING", protocol.REQUEST_STATUS_IN_PROCESS, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_PRE_ORDER_VALID", protocol.REQUEST_STATUS_RESERVED, protocol.TRANSACTION_STATUS_PRE_ORDER_VALID, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_NO_RECORD_FOUND", protocol.REQUEST_STATUS_NOT_FOUND, protocol.TRANSACTION_STATUS_NO_RECORD_FOUND, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_PRE_ORDER", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_PRE_ORDER, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_CONGESTION", protocol.REQUEST_STATUS_CONGESTION, protocol.TRANSACTION_STATUS_REJECTED_CONGESTION, protocol.EXECUTION_RESULT_RESERVED},
		{"TRANSACTION_STATUS_REJECTED_NODE_OUT_OF_SYNC", protocol.REQUEST_STATUS_OUT_OF_SYNC, protocol.TRANSACTION_STATUS_REJECTED_NODE_OUT_OF_SYNC, protocol.EXECUTION_RESULT_RESERVED},
	}
	for i := range tests {
		currTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(currTest.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, currTest.expect, translateTransactionStatusToRequestStatus(currTest.txStatus, currTest.exResult), fmt.Sprintf("%s was translated to %d", currTest.name, currTest.expect))
		})
	}
}

func TestResults_TranslateExecutionStatusToRequestStatus(t *testing.T) {
	tests := []struct {
		name     string
		expect   protocol.RequestStatus
		exResult protocol.ExecutionResult
	}{
		{"EXECUTION_RESULT_RESERVED", protocol.REQUEST_STATUS_RESERVED, protocol.EXECUTION_RESULT_RESERVED},
		{"EXECUTION_RESULT_SUCCESS", protocol.REQUEST_STATUS_COMPLETED, protocol.EXECUTION_RESULT_SUCCESS},
		{"EXECUTION_RESULT_ERROR_SMART_CONTRACT", protocol.REQUEST_STATUS_COMPLETED, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT},
		{"EXECUTION_RESULT_ERROR_INPUT", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.EXECUTION_RESULT_ERROR_INPUT},
		{"EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED", protocol.REQUEST_STATUS_BAD_REQUEST, protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED},
		{"EXECUTION_RESULT_ERROR_UNEXPECTED", protocol.REQUEST_STATUS_SYSTEM_ERROR, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED},
	}
	for i := range tests {
		currTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(currTest.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, currTest.expect, translateExecutionStatusToRequestStatus(currTest.exResult), fmt.Sprintf("%s was translated to %d", currTest.name, currTest.expect))
		})
	}
}

func TestResults_IsRequestValidChain(t *testing.T) {
	cfg := config.ForPublicApiTests(6, 0, 0)
	tx := builders.Transaction().WithVirtualChainId(6).Build().Transaction()
	txStatus, err := validateRequest(cfg, tx.ProtocolVersion(), tx.VirtualChainId())
	require.NoError(t, err, "there should be no error")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_RESERVED, txStatus, "virtual chain should be ok")
}

func TestResults_IsRequestValidChain_NonValid(t *testing.T) {
	cfg := config.ForPublicApiTests(44, 0, 0)
	tx := builders.Transaction().WithVirtualChainId(6).Build().Transaction()
	txStatus, err := validateRequest(cfg, tx.ProtocolVersion(), tx.VirtualChainId())
	require.Error(t, err, "there should be an error")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH, txStatus, "virtual chain should be wrong")
}

func TestResults_IsOutputPotentiallyOutOfSync(t *testing.T) {
	cfg := config.ForPublicApiTests(22, 0, time.Minute)
	var refBlockTimestamp primitives.TimestampNano
	require.False(t, isOutputPotentiallyOutOfSync(cfg, refBlockTimestamp), "empty block reference should not be out of sync")
	refBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano())
	require.False(t, isOutputPotentiallyOutOfSync(cfg, refBlockTimestamp), "recent block reference should not be out of sync")
	refBlockTimestamp = primitives.TimestampNano(time.Now().Add(time.Hour * -1).UnixNano())
	require.True(t, isOutputPotentiallyOutOfSync(cfg, refBlockTimestamp), "old block reference should be out of sync")
	refBlockTimestamp = primitives.TimestampNano(time.Now().Add(time.Hour).UnixNano())
	require.False(t, isOutputPotentiallyOutOfSync(cfg, refBlockTimestamp), "future block reference should not be out of sync")
}
