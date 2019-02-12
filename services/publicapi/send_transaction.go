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

	if input.ClientRequest == nil {
		err := errors.Errorf("client request is nil")
		s.logger.Info("send transaction received missing input", log.Error(err))
		return nil, err
	}

	tx := input.ClientRequest.SignedTransaction().Transaction()
	txHash := digest.CalcTxHash(tx)
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.Transaction(txHash), log.String("flow", "checkpoint"))

	if txStatus, err := validateRequest(s.config, tx.ProtocolVersion(), tx.VirtualChainId()); err != nil {
		logger.Info("send transaction received input failed", log.Error(err))
		return toSendTxOutput(&txOutput{transactionStatus: txStatus}), err
	}

	start := time.Now()
	defer s.metrics.sendTransactionTime.RecordSince(start)

	waitResult := s.waiter.add(txHash.KeyForMap())

	addResp, err := s.transactionPool.AddNewTransaction(ctx, &services.AddNewTransactionInput{
		SignedTransaction: input.ClientRequest.SignedTransaction(),
	})
	if err != nil {
		s.waiter.deleteByChannel(waitResult)
		logger.Info("adding transaction to TransactionPool failed", log.Error(err))
		return toSendTxOutput(addOutputToTxOutput(addResp)), err
	}

	if addResp.TransactionStatus == protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED {
		s.waiter.deleteByChannel(waitResult)
		return toSendTxOutput(addOutputToTxOutput(addResp)), nil
	}

	if input.ReturnImmediately != 0 {
		s.waiter.deleteByChannel(waitResult)
		return toSendTxOutput(addOutputToTxOutput(addResp)), nil
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.PublicApiSendTransactionTimeout())
	defer cancel()

	obj, err := s.waiter.wait(ctx, waitResult)
	if err != nil {
		logger.Info("waiting for transaction to be processed failed")
		return toSendTxOutput(addOutputToTxOutput(addResp)), err
	}
	return toSendTxOutput(obj.(*txOutput)), nil
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
