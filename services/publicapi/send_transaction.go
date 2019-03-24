// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

func (s *service) SendTransaction(parentCtx context.Context, input *services.SendTransactionInput) (*services.SendTransactionOutput, error) {
	ctx := trace.NewContext(parentCtx, "PublicApi.SendTransaction")
	start := time.Now()
	out, err := s.sendTransaction(ctx, input.ClientRequest, false)
	if out == nil {
		return nil, err
	}
	if out.transactionStatus == protocol.TRANSACTION_STATUS_COMMITTED {
		s.metrics.sendTransactionTime.RecordSince(start)
	}
	return toSendTxOutput(out), err
}

func (s *service) SendTransactionAsync(parentCtx context.Context, input *services.SendTransactionInput) (*services.SendTransactionOutput, error) {
	ctx := trace.NewContext(parentCtx, "PublicApi.SendTransactionAsync")
	out, err := s.sendTransaction(ctx, input.ClientRequest, true)
	if out == nil {
		return nil, err
	}
	return toSendTxOutput(out), err
}

func (s *service) sendTransaction(ctx context.Context, request *client.SendTransactionRequest, asyncMode bool) (*txOutput, error) {
	s.metrics.totalTransactionsFromClients.Inc()
	if request == nil {
		s.metrics.totalTransactionsErrNilRequest.Inc()
		err := errors.Errorf("client request is nil")
		s.logger.Info("send transaction received missing input", log.Error(err))
		return nil, err
	}

	tx := request.SignedTransaction().Transaction()
	txHash := digest.CalcTxHash(tx)
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.Transaction(txHash), log.String("flow", "checkpoint"))

	if txStatus, err := validateRequest(s.config, tx.ProtocolVersion(), tx.VirtualChainId()); err != nil {
		s.metrics.totalTransactionsErrInvalidRequest.Inc()
		logger.Info("send transaction received input failed", log.Error(err))
		return &txOutput{transactionStatus: txStatus}, err
	}

	logger.Info("send transaction request received")

	waitResult := s.waiter.add(txHash.KeyForMap())

	addResp, err := s.transactionPool.AddNewTransaction(ctx, &services.AddNewTransactionInput{
		SignedTransaction: request.SignedTransaction(),
	})
	if err != nil {
		s.metrics.totalTransactionsErrAddingToTxPool.Inc()
		s.waiter.deleteByChannel(waitResult)
		logger.Info("adding transaction to TransactionPool failed", log.Error(err))
		return addOutputToTxOutput(addResp), err
	}

	if addResp.TransactionStatus == protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED {
		s.metrics.totalTransactionsErrDuplicate.Inc()
		s.waiter.deleteByChannel(waitResult)
		return addOutputToTxOutput(addResp), nil
	}

	if asyncMode {
		s.waiter.deleteByChannel(waitResult)
		return addOutputToTxOutput(addResp), nil
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.PublicApiSendTransactionTimeout())
	defer cancel()

	obj, err := s.waiter.wait(ctx, waitResult)
	if err != nil {
		logger.Info("waiting for transaction to be processed failed")
		return addOutputToTxOutput(addResp), err
	}
	return obj.(*txOutput), nil
}

func addOutputToTxOutput(t *services.AddNewTransactionOutput) *txOutput {
	return &txOutput{
		transactionStatus:  t.TransactionStatus,
		transactionReceipt: t.TransactionReceipt,
		blockHeight:        t.BlockHeight,
		blockTimestamp:     t.BlockTimestamp,
	}
}

func toSendTxOutput(out *txOutput) *services.SendTransactionOutput {
	response := &client.SendTransactionResponseBuilder{
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
	return &services.SendTransactionOutput{ClientResponse: response.Build()}
}
