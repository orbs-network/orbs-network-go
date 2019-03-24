// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

type txOutput struct {
	transactionStatus  protocol.TransactionStatus
	transactionReceipt *protocol.TransactionReceipt
	blockHeight        primitives.BlockHeight
	blockTimestamp     primitives.TimestampNano
}

type queryOutput struct {
	requestStatus protocol.RequestStatus
	callOutput    *services.ProcessQueryOutput
}

func isOutputPotentiallyOutOfSync(config config.PublicApiConfig, referenceBlockTimestamp primitives.TimestampNano) bool {
	if referenceBlockTimestamp == 0 {
		return false
	}
	threshold := primitives.TimestampNano(time.Now().Add(config.PublicApiNodeSyncWarningTime() * -1).UnixNano())
	return threshold > referenceBlockTimestamp
}

func validateRequest(config config.PublicApiConfig, protocolVersion primitives.ProtocolVersion, vcId primitives.VirtualChainId) (protocol.TransactionStatus, error) {
	if primitives.ProtocolVersion(1) != protocolVersion {
		return protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION, errors.Errorf("invalid protocol version %d", protocolVersion)
	}

	if config.VirtualChainId() != vcId {
		return protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH, errors.Errorf("virtual chain mismatch received %d but expected %d", vcId, config.VirtualChainId())
	}

	return protocol.TRANSACTION_STATUS_RESERVED, nil
}

func translateTransactionStatusToRequestStatus(txStatus protocol.TransactionStatus, executionResult protocol.ExecutionResult) protocol.RequestStatus {
	switch txStatus {
	case protocol.TRANSACTION_STATUS_COMMITTED:
		return translateExecutionStatusToRequestStatus(executionResult)
	case protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED:
		return translateExecutionStatusToRequestStatus(executionResult)
	case protocol.TRANSACTION_STATUS_PENDING:
		return protocol.REQUEST_STATUS_IN_PROCESS
	case protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING:
		return protocol.REQUEST_STATUS_IN_PROCESS
	case protocol.TRANSACTION_STATUS_NO_RECORD_FOUND:
		return protocol.REQUEST_STATUS_NOT_FOUND
	case protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_PRE_ORDER:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.TRANSACTION_STATUS_REJECTED_CONGESTION:
		return protocol.REQUEST_STATUS_CONGESTION
	case protocol.TRANSACTION_STATUS_REJECTED_NODE_OUT_OF_SYNC:
		return protocol.REQUEST_STATUS_OUT_OF_SYNC
	}
	return protocol.REQUEST_STATUS_RESERVED
}

func translateExecutionStatusToRequestStatus(executionResult protocol.ExecutionResult) protocol.RequestStatus {
	switch executionResult {
	case protocol.EXECUTION_RESULT_SUCCESS:
		return protocol.REQUEST_STATUS_COMPLETED
	case protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT:
		return protocol.REQUEST_STATUS_COMPLETED
	case protocol.EXECUTION_RESULT_ERROR_INPUT:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED:
		return protocol.REQUEST_STATUS_BAD_REQUEST
	case protocol.EXECUTION_RESULT_ERROR_UNEXPECTED:
		return protocol.REQUEST_STATUS_SYSTEM_ERROR
	}
	return protocol.REQUEST_STATUS_RESERVED
}
