// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

func (s *service) GetTransactionStatus(parentCtx context.Context, input *services.GetTransactionStatusInput) (*services.GetTransactionStatusOutput, error) {
	ctx := trace.NewContext(parentCtx, "PublicApi.GetTransactionStatus")

	if input.ClientRequest == nil {
		err := errors.Errorf("client request is nil")
		s.logger.Info("get transaction status received missing input", log.Error(err))
		return nil, err
	}

	tx := input.ClientRequest.TransactionRef()
	txHash := tx.Txhash()
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.Transaction(txHash), log.String("flow", "checkpoint"))

	if txStatus, err := validateRequest(s.config, tx.ProtocolVersion(), tx.VirtualChainId()); err != nil {
		logger.Info("get transaction status received input failed", log.Error(err))
		return toGetTxStatusOutput(s.config, &txOutput{transactionStatus: txStatus}), err
	}

	logger.Info("get transaction status request received")

	return s.getTransactionStatus(ctx, s.config, txHash, tx.TransactionTimestamp())
}

func (s *service) getTransactionStatus(ctx context.Context, config config.PublicApiConfig, txHash primitives.Sha256, txTimestamp primitives.TimestampNano) (*services.GetTransactionStatusOutput, error) {
	start := time.Now()
	defer s.metrics.getTransactionStatusTime.RecordSince(start)

	if txReceipt, err, done := s.getFromTxPool(ctx, txHash, txTimestamp); done {
		return toGetTxStatusOutput(config, txReceipt), err
	}

	blockReceipt, err := s.getFromBlockStorage(ctx, txHash, txTimestamp)
	if err != nil {
		return nil, err
	}
	return toGetTxStatusOutput(config, blockReceipt), err
}

func (s *service) getFromTxPool(ctx context.Context, txHash primitives.Sha256, timestamp primitives.TimestampNano) (*txOutput, error, bool) {
	txReceipt, err := s.transactionPool.GetCommittedTransactionReceipt(ctx, &services.GetCommittedTransactionReceiptInput{
		Txhash:               txHash,
		TransactionTimestamp: timestamp,
	})
	if err != nil {
		s.logger.Info("get transaction txStatus failed in transactionPool", log.Error(err), log.String("flow", "checkpoint"), log.Transaction(txHash))
		return poolOutputToTxOutput(txReceipt), err, true
	}
	if txReceipt.TransactionStatus == protocol.TRANSACTION_STATUS_PENDING || txReceipt.TransactionStatus == protocol.TRANSACTION_STATUS_COMMITTED {
		return poolOutputToTxOutput(txReceipt), nil, true
	}
	return nil, nil, false
}

func poolOutputToTxOutput(t *services.GetCommittedTransactionReceiptOutput) *txOutput {
	return &txOutput{
		transactionStatus:  t.TransactionStatus,
		transactionReceipt: t.TransactionReceipt,
		blockHeight:        t.BlockHeight,
		blockTimestamp:     t.BlockTimestamp,
	}
}

func (s *service) getFromBlockStorage(ctx context.Context, txHash primitives.Sha256, timestamp primitives.TimestampNano) (*txOutput, error) {
	txReceipt, err := s.blockStorage.GetTransactionReceipt(ctx, &services.GetTransactionReceiptInput{
		Txhash:               txHash,
		TransactionTimestamp: timestamp,
	})
	if err != nil {
		s.logger.Info("get transaction txStatus failed in blockStorage", log.Error(err), log.String("flow", "checkpoint"), log.Transaction(txHash))
		return nil, err
	}
	return blockOutputToTxOutput(txReceipt), nil
}

func blockOutputToTxOutput(t *services.GetTransactionReceiptOutput) *txOutput {
	txStatus := protocol.TRANSACTION_STATUS_NO_RECORD_FOUND
	if t.TransactionReceipt != nil {
		txStatus = protocol.TRANSACTION_STATUS_COMMITTED
	}
	return &txOutput{
		transactionStatus:  txStatus,
		transactionReceipt: t.TransactionReceipt,
		blockHeight:        t.BlockHeight,
		blockTimestamp:     t.BlockTimestamp,
	}
}

func toGetTxStatusOutput(config config.PublicApiConfig, out *txOutput) *services.GetTransactionStatusOutput {
	response := &client.GetTransactionStatusResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  translateTransactionStatusToRequestStatus(out.transactionStatus, protocol.EXECUTION_RESULT_RESERVED),
			BlockHeight:    out.blockHeight,
			BlockTimestamp: out.blockTimestamp,
		},
		TransactionStatus:  out.transactionStatus,
		TransactionReceipt: nil,
	}
	if out.transactionReceipt != nil {
		response.RequestResult.RequestStatus = translateTransactionStatusToRequestStatus(out.transactionStatus, out.transactionReceipt.ExecutionResult())
		response.TransactionReceipt = protocol.TransactionReceiptBuilderFromRaw(out.transactionReceipt.Raw())
	}
	if response.TransactionReceipt == nil && isOutputPotentiallyOutOfSync(config, out.blockTimestamp) {
		response.RequestResult.RequestStatus = protocol.REQUEST_STATUS_OUT_OF_SYNC
	}
	return &services.GetTransactionStatusOutput{ClientResponse: response.Build()}
}
