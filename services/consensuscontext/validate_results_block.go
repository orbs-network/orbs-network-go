// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"bytes"
	"context"
	"fmt"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/crypto-lib-go/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

type rxValidator func(ctx context.Context, vcrx *rxValidatorContext) error

// TODO v3 consider changing the way the validator works not to have a "context" see issue https://github.com/orbs-network/orbs-network-go/issues/1555
type rxValidatorContext struct {
	virtualChainId         primitives.VirtualChainId
	input                  *services.ValidateResultsBlockInput
	fixedPrevRefTime       primitives.TimestampSeconds
	getStateHash           func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error)
	processTransactionSet  func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error)
	calcReceiptsMerkleRoot func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
	calcStateDiffHash      func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

func validateRxProtocolVersion(ctx context.Context, vcrx *rxValidatorContext) error {
	txPv := vcrx.input.TransactionsBlock.Header.ProtocolVersion()
	rxPv := vcrx.input.ResultsBlock.Header.ProtocolVersion()
	if rxPv != txPv {
		return errors.Wrapf(ErrMismatchedProtocolVersion, "mismatched protocol version between transactions and results tx %v rx %v", txPv, rxPv)
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

func validateRxBlockProposer(ctx context.Context, vcrx *rxValidatorContext) error {
	blockProposer := vcrx.input.ResultsBlock.Header.BlockProposerAddress()
	if len(blockProposer) > 0 { // If rx block header - block proposer is len 0 this is older version and for backward compatibility validate check is skipped
		expectedBlockProposer := vcrx.input.BlockProposerAddress
		if !bytes.Equal(blockProposer, expectedBlockProposer) {
			return errors.Wrapf(ErrMismatchedBlockProposer, "Results Block expected %v actual %v", expectedBlockProposer, blockProposer)
		}
		txBlockProposer := vcrx.input.TransactionsBlock.Header.BlockProposerAddress()
		if !bytes.Equal(blockProposer, txBlockProposer) {
			return errors.Wrapf(ErrMismatchedTxRxBlockProposer, "txBlock %v rxBlock %v", txBlockProposer, blockProposer)
		}
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

func validateIdenticalTxRxBlockReferenceTimes(ctx context.Context, vcrx *rxValidatorContext) error {
	txRefTime := vcrx.input.TransactionsBlock.Header.ReferenceTime()
	rxRefTime := vcrx.input.ResultsBlock.Header.ReferenceTime()
	if rxRefTime != txRefTime {
		return errors.Wrapf(ErrMismatchedTxRxBlockRefTimes, "txRefTime %v rxRefTime %v", txRefTime, rxRefTime)
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
	// Validate transaction execution
	// Execute the ordered transactions set by calling VirtualMachine.ProcessTransactionSet creating receipts and state diff. Using the provided header timestamp as a reference timestamp.
	processTxsOut, err := vcrx.processTransactionSet(ctx, &services.ProcessTransactionSetInput{
		CurrentBlockHeight:        vcrx.input.TransactionsBlock.Header.BlockHeight(),
		CurrentBlockTimestamp:     vcrx.input.TransactionsBlock.Header.Timestamp(),
		SignedTransactions:        vcrx.input.TransactionsBlock.SignedTransactions,
		BlockProposerAddress:      vcrx.input.BlockProposerAddress,
		CurrentBlockReferenceTime: vcrx.input.TransactionsBlock.Header.ReferenceTime(),
		PrevBlockReferenceTime:    vcrx.fixedPrevRefTime,
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
		diff := compare(vcrx.input.ResultsBlock.ContractStateDiffs, processTxsOut.ContractStateDiffs)
		return errors.Wrapf(validators.ErrMismatchedStateDiffHash, "expectedStateDiffHash %v calculatedStateDiffHash %v, countMismatches %d, mismatches %#v", expectedStateDiffHash, calculatedStateDiffHash, len(diff), diff)
	}

	return nil
}

func compare(expectedDiffs []*protocol.ContractStateDiff, calculatedDiffs []*protocol.ContractStateDiff) map[string]string {
	diff := map[string]string{}

	expectedMap := map[string]*protocol.StateRecord{}
	for _, expectedCsd := range expectedDiffs {
		itr := expectedCsd.StateDiffsIterator()
		for itr.HasNext() {
			record := itr.NextStateDiffs()
			expectedMap[expectedCsd.ContractName().KeyForMap()+"/"+record.StringKey()] = record
		}
	}

	for _, calculatedCsd := range calculatedDiffs {
		itr := calculatedCsd.StateDiffsIterator()
		for itr.HasNext() {
			record := itr.NextStateDiffs()
			contractRecordKey := calculatedCsd.ContractName().KeyForMap() + "/" + record.StringKey()
			expectedValue, expected := expectedMap[contractRecordKey]
			if expected {
				if !bytes.Equal(expectedValue.Raw(), record.Raw()) {
					diff[contractRecordKey] = fmt.Sprintf("e: %s <==> c: %s", expectedValue.StringValue(), record.StringValue())
				}
				delete(expectedMap, contractRecordKey)
			} else {
				diff[contractRecordKey] = fmt.Sprintf("e: NA <==> c: %s", record.StringValue())
			}
		}
	}

	for key, record := range expectedMap {
		diff[key] = fmt.Sprintf("e: %s <==> c: NA", record.StringValue())
	}

	return diff
}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	prevBlockReferenceTime, err := s.prevReferenceOrGenesis(ctx, input.CurrentBlockHeight, input.PrevBlockReferenceTime)
	if err != nil {
		return &services.ValidateResultsBlockOutput{}, errors.Wrapf(ErrFailedGenesisRefTime, "ValidateResultsBlock failed genesis time %s", err)
	}

	vcrx := &rxValidatorContext{
		virtualChainId:         s.config.VirtualChainId(),
		input:                  input,
		getStateHash:           s.stateStorage.GetStateHash,
		processTransactionSet:  s.virtualMachine.ProcessTransactionSet,
		calcReceiptsMerkleRoot: digest.CalcReceiptsMerkleRoot,
		calcStateDiffHash:      digest.CalcStateDiffHash,
		fixedPrevRefTime:       prevBlockReferenceTime,
	}

	validators := []rxValidator{
		validateRxProtocolVersion,
		validateRxVirtualChainID,
		validateRxBlockHeight,
		validateRxTxBlockPtrMatchesActualTxBlock,
		validateIdenticalTxRxTimestamp,
		validateIdenticalTxRxBlockReferenceTimes,
		validateRxPrevBlockHashPtr,
		validateRxBlockProposer,
		validateReceiptsMerkleRoot,
		validateRxStateDiffHash,
		validatePreExecutionStateMerkleRoot,
		validateExecution,
	}

	for _, v := range validators {
		if ctx.Err() != nil {
			return &services.ValidateResultsBlockOutput{}, errors.New("context canceled")
		}

		if err := v(ctx, vcrx); err != nil {
			return &services.ValidateResultsBlockOutput{}, err
		}
	}
	return &services.ValidateResultsBlockOutput{}, nil
}
