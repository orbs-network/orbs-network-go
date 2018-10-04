package publicapi

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) GetTransactionStatus(input *services.GetTransactionStatusInput) (*services.GetTransactionStatusOutput, error) {
	if input.ClientRequest == nil {
		err := errors.Errorf("error: missing input (client request is nil)")
		s.logger.Info("get transaction status received missing input", log.Error(err))
		return nil, err
	}

	s.logger.Info("get transaction status request received", log.String("flow", "checkpoint"), log.Stringable("txHash", input.ClientRequest.Txhash()))
	txReceipt, err := s.transactionPool.GetCommittedTransactionReceipt(&services.GetCommittedTransactionReceiptInput{
		Txhash:               input.ClientRequest.Txhash(),
		TransactionTimestamp: input.ClientRequest.TransactionTimestamp(),
	})
	if err != nil {
		s.logger.Info("get transaction status failed in transactionPool", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", input.ClientRequest.Txhash()))
		return toGetTxOutput(txStatusToTxResponse(txReceipt)), err
	}
	if txReceipt.TransactionStatus != protocol.TRANSACTION_STATUS_NO_RECORD_FOUND {
		return toGetTxOutput(txStatusToTxResponse(txReceipt)), nil
	}

	blockReceipt, err := s.blockStorage.GetTransactionReceipt(&services.GetTransactionReceiptInput{
		Txhash:               input.ClientRequest.Txhash(),
		TransactionTimestamp: input.ClientRequest.TransactionTimestamp(),
	})
	if err != nil {
		s.logger.Info("get transaction status failed in blockStorage", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", input.ClientRequest.Txhash()))
		return toGetTxOutput(blockToTxResponse(blockReceipt)), err
	}
	return toGetTxOutput(blockToTxResponse(blockReceipt)), nil
}

func txStatusToTxResponse(txStatus *services.GetCommittedTransactionReceiptOutput) *txResponse {
	return &txResponse{
		transactionStatus:  txStatus.TransactionStatus,
		transactionReceipt: txStatus.TransactionReceipt,
		blockHeight:        txStatus.BlockHeight,
		blockTimestamp:     txStatus.BlockTimestamp,
	}
}

func blockToTxResponse(bReceipt *services.GetTransactionReceiptOutput) *txResponse {
	status := protocol.TRANSACTION_STATUS_NO_RECORD_FOUND
	if bReceipt.TransactionReceipt != nil {
		status = protocol.TRANSACTION_STATUS_COMMITTED
	}
	return &txResponse{
		transactionStatus:  status,
		transactionReceipt: bReceipt.TransactionReceipt,
		blockHeight:        bReceipt.BlockHeight,
		blockTimestamp:     bReceipt.BlockTimestamp,
	}
}

func toGetTxOutput(transactionOutput *txResponse) *services.GetTransactionStatusOutput {
	var receiptForClient *protocol.TransactionReceiptBuilder = nil

	if receipt := transactionOutput.transactionReceipt; receipt != nil {
		receiptForClient = &protocol.TransactionReceiptBuilder{
			Txhash:              receipt.Txhash(),
			ExecutionResult:     receipt.ExecutionResult(),
			OutputArgumentArray: receipt.OutputArgumentArray(),
		}
	}

	response := &client.GetTransactionStatusResponseBuilder{
		RequestStatus:      translateTxStatusToResponseCode(transactionOutput.transactionStatus),
		TransactionReceipt: receiptForClient,
		TransactionStatus:  transactionOutput.transactionStatus,
		BlockHeight:        transactionOutput.blockHeight,
		BlockTimestamp:     transactionOutput.blockTimestamp,
	}

	return &services.GetTransactionStatusOutput{ClientResponse: response.Build()}
}
