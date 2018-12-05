package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) GetTransactionReceiptProof(ctx context.Context, input *services.GetTransactionReceiptProofInput) (*services.GetTransactionReceiptProofOutput, error) {
	if input.ClientRequest == nil {
		err := errors.Errorf("error: missing input (client request is nil)")
		s.logger.Info("get transaction status received missing input", log.Error(err))
		return nil, err
	}

	txHash := input.ClientRequest.Txhash()

	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.Transaction(txHash), log.String("flow", "get-proof"))

	if txStatus := isTransactionValuesValid(s.config, input.ClientRequest.VirtualChainId(), input.ClientRequest.ProtocolVersion()); txStatus != protocol.TRANSACTION_STATUS_RESERVED {
		err := errors.Errorf("error input %s", txStatus.String())
		logger.Info("get transaction receipt proof input failed", log.Error(err))
		return toGetReceiptOutput(nil, nil), err
	}

	logger.Info("get transaction receipt proof request received")

	status, err := s.getTransactionStatus(ctx, txHash, input.ClientRequest.TransactionTimestamp())
	if err != nil || status == nil || status.ClientResponse.TransactionStatus() != protocol.TRANSACTION_STATUS_COMMITTED {
		if err != nil || status == nil {
			logger.Info("get transaction receipt proof failed to get transaction status", log.Error(err))
		} else {
			logger.Info("get transaction receipt proof failed: transaction status not commit", log.Stringable("tx status", status.ClientResponse.TransactionStatus()))
		}
		return toGetReceiptOutput(status, nil), err
	}

	receiptProof, err := s.blockStorage.GenerateReceiptProof(ctx, &services.GenerateReceiptProofInput{
		Txhash:      txHash,
		BlockHeight: status.ClientResponse.BlockHeight(),
	})

	if err != nil {
		logger.Info("get transaction receipt proof failed to get block proof", log.Error(err))
		return toGetReceiptOutput(status, nil), err
	}

	return toGetReceiptOutput(status, receiptProof), nil
}

func toGetReceiptOutput(status *services.GetTransactionStatusOutput, output *services.GenerateReceiptProofOutput) *services.GetTransactionReceiptProofOutput {
	txStatus := protocol.TRANSACTION_STATUS_NO_RECORD_FOUND
	var txBlock primitives.BlockHeight
	var txTime primitives.TimestampNano

	if status != nil {
		txStatus = status.ClientResponse.TransactionStatus()
		txBlock = status.ClientResponse.BlockHeight()
		txTime = status.ClientResponse.BlockTimestamp()
	}

	var proofForClient *protocol.ReceiptProofBuilder = nil

	if output != nil {
		proofForClient = &protocol.ReceiptProofBuilder{
			Header:       nil,
			BlockProof:   nil,
			ReceiptProof: nil,
			ReceiptIndex: nil,
			Receipt:      nil,
		}
	}

	// TODO issue 67 PROOF get raw info
	response := client.GetTransactionReceiptProofResponseBuilder{
		RequestStatus:     translateTxStatusToResponseCode(txStatus),
		Proof:             proofForClient,
		TransactionStatus: txStatus,
		BlockHeight:       txBlock,
		BlockTimestamp:    txTime,
	}

	return &services.GetTransactionReceiptProofOutput{ClientResponse: response.Build()}
}
