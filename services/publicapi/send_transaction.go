package publicapi

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

func (s *service) SendTransaction(ctx context.Context, input *services.SendTransactionInput) (*services.SendTransactionOutput, error) {
	if input.ClientRequest == nil {
		err := errors.Errorf("error missing input (client request is nil)")
		s.logger.Info("send transaction received missing input", log.Error(err))
		return nil, err
	}

	tx := input.ClientRequest.SignedTransaction()
	if txStatus := isTransactionRequestValid(s.config, tx.Transaction()); txStatus != protocol.TRANSACTION_STATUS_RESERVED {
		err := errors.Errorf("error input %s", txStatus.String())
		s.logger.Info("send transaction received input failed", log.Error(err))
		return toSendTxOutput(&txResponse{transactionStatus: txStatus}), err
	}

	txHash := digest.CalcTxHash(tx.Transaction())
	s.logger.Info("send transaction request received", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))

	start := time.Now()
	defer s.metrics.sendTransaction.RecordSince(start)

	waitResult := s.waiter.add(txHash.KeyForMap())

	addResp, err := s.transactionPool.AddNewTransaction(ctx, &services.AddNewTransactionInput{SignedTransaction: tx})
	if err != nil {
		s.waiter.deleteByChannel(waitResult)
		s.logger.Info("adding transaction to TransactionPool failed", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
		return toSendTxOutput(toTxResponse(addResp)), errors.Wrap(err, fmt.Sprintf("error '%s' for transaction result", addResp))
	}

	if addResp.TransactionStatus == protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED {
		s.waiter.deleteByChannel(waitResult)
		return toSendTxOutput(toTxResponse(addResp)), nil
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.SendTransactionTimeout())
	defer cancel()

	obj, err := s.waiter.wait(ctx, waitResult)
	if err != nil {
		s.logger.Info("waiting for transaction to be processed failed", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
		return toSendTxOutput(toTxResponse(addResp)), err
	}
	return toSendTxOutput(obj.(*txResponse)), nil
}

func toTxResponse(t *services.AddNewTransactionOutput) *txResponse {
	return &txResponse{
		transactionStatus:  t.TransactionStatus,
		transactionReceipt: t.TransactionReceipt,
		blockHeight:        t.BlockHeight,
		blockTimestamp:     t.BlockTimestamp,
	}
}

func toSendTxOutput(transactionOutput *txResponse) *services.SendTransactionOutput {
	var receiptForClient *protocol.TransactionReceiptBuilder = nil

	if receipt := transactionOutput.transactionReceipt; receipt != nil {
		receiptForClient = &protocol.TransactionReceiptBuilder{
			Txhash:              receipt.Txhash(),
			ExecutionResult:     receipt.ExecutionResult(),
			OutputArgumentArray: receipt.OutputArgumentArray(),
		}
	}

	response := &client.SendTransactionResponseBuilder{
		RequestStatus:      translateTxStatusToResponseCode(transactionOutput.transactionStatus),
		TransactionReceipt: receiptForClient,
		TransactionStatus:  transactionOutput.transactionStatus,
		BlockHeight:        transactionOutput.blockHeight,
		BlockTimestamp:     transactionOutput.blockTimestamp,
	}

	return &services.SendTransactionOutput{ClientResponse: response.Build()}
}
