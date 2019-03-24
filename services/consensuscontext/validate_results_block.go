// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

type rxValidator func(ctx context.Context, vcrx *rxValidatorContext) error

type rxValidatorContext struct {
	protocolVersion        primitives.ProtocolVersion
	virtualChainId         primitives.VirtualChainId
	input                  *services.ValidateResultsBlockInput
	getStateHash           func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error)
	processTransactionSet  func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error)
	calcReceiptsMerkleRoot func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
	calcStateDiffHash      func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

func validateRxProtocolVersion(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedProtocolVersion := vcrx.protocolVersion
	checkedProtocolVersion := vcrx.input.ResultsBlock.Header.ProtocolVersion()
	if checkedProtocolVersion != expectedProtocolVersion {
		return errors.Wrapf(ErrMismatchedProtocolVersion, "expected %v actual %v", expectedProtocolVersion, checkedProtocolVersion)
	}
	return nil
}

func validateRxVirtualChainID(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedVirtualChainId := vcrx.virtualChainId
	checkedVirtualChainId := vcrx.input.ResultsBlock.Header.VirtualChainId()
	if checkedVirtualChainId != expectedVirtualChainId {
		return errors.Wrapf(ErrMismatchedVirtualChainID, "expected %v actual %v", expectedVirtualChainId, checkedVirtualChainId)
	}
	return nil
}

func validateRxBlockHeight(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedBlockHeight := vcrx.input.CurrentBlockHeight
	checkedBlockHeight := vcrx.input.ResultsBlock.Header.BlockHeight()
	if checkedBlockHeight != expectedBlockHeight {
		return errors.Wrapf(ErrMismatchedBlockHeight, "expected %v actual %v", expectedBlockHeight, checkedBlockHeight)
	}
	txBlockHeight := vcrx.input.TransactionsBlock.Header.BlockHeight()
	if checkedBlockHeight != txBlockHeight {
		return errors.Wrapf(ErrMismatchedTxRxBlockHeight, "txBlock %v rxBlock %v", txBlockHeight, checkedBlockHeight)
	}
	return nil
}

func validateRxTxBlockPtrMatchesActualTxBlock(ctx context.Context, vcrx *rxValidatorContext) error {
	txBlockHashPtr := vcrx.input.ResultsBlock.Header.TransactionsBlockHashPtr()
	expectedTxBlockHashPtr := digest.CalcTransactionsBlockHash(vcrx.input.TransactionsBlock)
	if !bytes.Equal(txBlockHashPtr, expectedTxBlockHashPtr) {
		return errors.Wrapf(ErrMismatchedTxHashPtrToActualTxBlock, "expected %v actual %v", expectedTxBlockHashPtr, txBlockHashPtr)
	}
	return nil
}

func validateIdenticalTxRxTimestamp(ctx context.Context, vcrx *rxValidatorContext) error {
	txTimestamp := vcrx.input.TransactionsBlock.Header.Timestamp()
	rxTimestamp := vcrx.input.ResultsBlock.Header.Timestamp()
	if rxTimestamp != txTimestamp {
		return errors.Wrapf(ErrMismatchedTxRxTimestamps, "txTimestamp %v rxTimestamp %v", txTimestamp, rxTimestamp)
	}
	return nil
}

func validateRxPrevBlockHashPtr(ctx context.Context, vcrx *rxValidatorContext) error {
	prevBlockHashPtr := vcrx.input.ResultsBlock.Header.PrevBlockHashPtr()
	expectedPrevBlockHashPtr := vcrx.input.PrevBlockHash
	if !bytes.Equal(prevBlockHashPtr, expectedPrevBlockHashPtr) {
		return errors.Wrapf(ErrMismatchedPrevBlockHash, "expected %v actual %v", expectedPrevBlockHashPtr, prevBlockHashPtr)
	}
	return nil
}

func validateReceiptsMerkleRoot(ctx context.Context, vcrx *rxValidatorContext) error {
	return validators.ValidateReceiptsMerkleRoot(&validators.BlockValidatorContext{
		TransactionsBlock:      vcrx.input.TransactionsBlock,
		ResultsBlock:           vcrx.input.ResultsBlock,
		CalcReceiptsMerkleRoot: vcrx.calcReceiptsMerkleRoot,
	})
}

func validateRxStateDiffHash(ctx context.Context, vcrx *rxValidatorContext) error {
	return validators.ValidateResultsBlockStateDiffHash(&validators.BlockValidatorContext{
		TransactionsBlock: vcrx.input.TransactionsBlock,
		ResultsBlock:      vcrx.input.ResultsBlock,
		CalcStateDiffHash: vcrx.calcStateDiffHash,
	})
}

func validatePreExecutionStateMerkleRoot(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedPreExecutionMerkleRoot := vcrx.input.ResultsBlock.Header.PreExecutionStateMerkleRootHash()
	getStateHashOut, err := vcrx.getStateHash(ctx, &services.GetStateHashInput{
		BlockHeight: vcrx.input.ResultsBlock.Header.BlockHeight() - 1,
	})
	if err != nil {
		return errors.Wrapf(ErrGetStateHash, "ValidateResultsBlock.validatePreExecutionStateMerkleRoot() error from GetStateHash(): %v", err)
	}
	if !bytes.Equal(expectedPreExecutionMerkleRoot, getStateHashOut.StateMerkleRootHash) {
		return errors.Wrapf(ErrMismatchedPreExecutionStateMerkleRoot, "expected %v actual %v", expectedPreExecutionMerkleRoot, getStateHashOut.StateMerkleRootHash)
	}
	return nil
}

func validateExecution(ctx context.Context, vcrx *rxValidatorContext) error {
	//Validate transaction execution
	// Execute the ordered transactions set by calling VirtualMachine.ProcessTransactionSet creating receipts and state diff. Using the provided header timestamp as a reference timestamp.
	processTxsOut, err := vcrx.processTransactionSet(ctx, &services.ProcessTransactionSetInput{
		CurrentBlockHeight:    vcrx.input.TransactionsBlock.Header.BlockHeight(),
		CurrentBlockTimestamp: vcrx.input.TransactionsBlock.Header.Timestamp(),
		SignedTransactions:    vcrx.input.TransactionsBlock.SignedTransactions,
	})
	if err != nil {
		return errors.Wrapf(ErrProcessTransactionSet, "ValidateResultsBlock.validateExecution() error from ProcessTransactionSet(): %v", err)
	}
	// Compare the receipts merkle root hash to the one in the block.
	expectedReceiptsMerkleRoot := vcrx.input.ResultsBlock.Header.ReceiptsMerkleRootHash()
	calculatedReceiptMerkleRoot, err := vcrx.calcReceiptsMerkleRoot(processTxsOut.TransactionReceipts)
	if err != nil {
		return errors.Wrapf(validators.ErrCalcReceiptsMerkleRoot, "ValidateResultsBlock error ProcessTransactionSet calcReceiptsMerkleRoot(): %v", err)
	}
	if !bytes.Equal(expectedReceiptsMerkleRoot, calculatedReceiptMerkleRoot) {
		return errors.Wrapf(validators.ErrMismatchedReceiptsRootHash, "expected %v actual %v", expectedReceiptsMerkleRoot, calculatedReceiptMerkleRoot)
	}

	// Compare the state diff hash to the one in the block (supports only deterministic execution).
	expectedStateDiffHash := vcrx.input.ResultsBlock.Header.StateDiffHash()
	calculatedStateDiffHash, err := vcrx.calcStateDiffHash(processTxsOut.ContractStateDiffs)
	if err != nil {
		return errors.Wrapf(validators.ErrCalcStateDiffHash, "ValidateResultsBlock error ProcessTransactionSet calculateStateDiffHash(): %v", err)
	}
	if !bytes.Equal(expectedStateDiffHash, calculatedStateDiffHash) {
		return errors.Wrapf(validators.ErrMismatchedStateDiffHash, "expected %v actual %v", expectedStateDiffHash, calculatedStateDiffHash)
	}

	return nil
}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {

	vcrx := &rxValidatorContext{
		protocolVersion:        s.config.ProtocolVersion(),
		virtualChainId:         s.config.VirtualChainId(),
		input:                  input,
		getStateHash:           s.stateStorage.GetStateHash,
		processTransactionSet:  s.virtualMachine.ProcessTransactionSet,
		calcReceiptsMerkleRoot: digest.CalcReceiptsMerkleRoot,
		calcStateDiffHash:      digest.CalcStateDiffHash,
	}

	validators := []rxValidator{
		validateRxProtocolVersion,
		validateRxVirtualChainID,
		validateRxBlockHeight,
		validateRxTxBlockPtrMatchesActualTxBlock,
		validateIdenticalTxRxTimestamp,
		validateRxPrevBlockHashPtr,
		validateReceiptsMerkleRoot,
		validateRxStateDiffHash,
		validatePreExecutionStateMerkleRoot,
		validateExecution,
	}

	for _, v := range validators {
		if err := v(ctx, vcrx); err != nil {
			return &services.ValidateResultsBlockOutput{}, err
		}
	}
	return &services.ValidateResultsBlockOutput{}, nil
}
