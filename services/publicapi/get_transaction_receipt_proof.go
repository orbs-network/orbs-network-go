// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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

func (s *service) GetTransactionReceiptProof(parentCtx context.Context, input *services.GetTransactionReceiptProofInput) (*services.GetTransactionReceiptProofOutput, error) {
	ctx := trace.NewContext(parentCtx, "PublicApi.GetTransactionReceiptProof")

	if input.ClientRequest == nil {
		err := errors.Errorf("client request is nil")
		s.logger.Info("get transaction receipt proof received missing input", log.Error(err))
		return nil, err
	}

	tx := input.ClientRequest.TransactionRef()
	txHash := tx.Txhash()
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.Transaction(txHash), log.String("flow", "checkpoint"))

	if txStatus, err := validateRequest(s.config, tx.ProtocolVersion(), tx.VirtualChainId()); err != nil {
		logger.Info("get transaction receipt proof received input failed", log.Error(err))
		return toGetTxProofOutput(toGetTxStatusOutput(s.config, &txOutput{transactionStatus: txStatus}), nil), err
	}

	logger.Info("get transaction receipt proof request received")

	txStatusOutput, err := s.getTransactionStatus(ctx, s.config, txHash, tx.TransactionTimestamp())
	if err != nil || txStatusOutput == nil || txStatusOutput.ClientResponse.TransactionStatus() != protocol.TRANSACTION_STATUS_COMMITTED {
		if err != nil || txStatusOutput == nil {
			logger.Info("get transaction receipt proof failed to get transaction txStatus", log.Error(err))
		} else {
			logger.Info("get transaction receipt proof failed: txStatus not committed", log.Stringable("tx-status", txStatusOutput.ClientResponse.TransactionStatus()))
		}
		return toGetTxProofOutput(txStatusOutput, nil), err
	}

	proofOutput, err := s.blockStorage.GenerateReceiptProof(ctx, &services.GenerateReceiptProofInput{
		Txhash:      txHash,
		BlockHeight: txStatusOutput.ClientResponse.RequestResult().BlockHeight(),
	})
	if err != nil {
		logger.Info("get transaction receipt proof failed to get block proof", log.Error(err))
		return toGetTxProofOutput(txStatusOutput, nil), err
	}

	return toGetTxProofOutput(txStatusOutput, proofOutput), nil
}

func toGetTxProofOutput(txStatusOutput *services.GetTransactionStatusOutput, proofOutput *services.GenerateReceiptProofOutput) *services.GetTransactionReceiptProofOutput {
	var txStatus protocol.TransactionStatus
	var requestResult *client.RequestResultBuilder
	var transactionReceipt *protocol.TransactionReceiptBuilder
	if txStatusOutput != nil {
		txStatus = txStatusOutput.ClientResponse.TransactionStatus()
		requestResult = client.RequestResultBuilderFromRaw(txStatusOutput.ClientResponse.RequestResult().Raw())
		transactionReceipt = protocol.TransactionReceiptBuilderFromRaw(txStatusOutput.ClientResponse.TransactionReceipt().Raw())
	} else {
		txStatus = protocol.TRANSACTION_STATUS_NO_RECORD_FOUND
		requestResult = &client.RequestResultBuilder{
			RequestStatus: translateTransactionStatusToRequestStatus(txStatus, protocol.EXECUTION_RESULT_RESERVED),
		}
	}

	var proofForClient primitives.PackedReceiptProof
	if proofOutput != nil {
		proofForClient = proofOutput.Proof.Raw()
	}

	response := client.GetTransactionReceiptProofResponseBuilder{
		RequestResult:      requestResult,
		TransactionStatus:  txStatus,
		TransactionReceipt: transactionReceipt,
		PackedProof:        proofForClient,
	}

	return &services.GetTransactionReceiptProofOutput{ClientResponse: response.Build()}
}
