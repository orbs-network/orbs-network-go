// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"

	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type blockValidator func(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error

type validatorContext struct {
	blockHash              primitives.Sha256
	CalcReceiptsMerkleRoot func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
	CalcStateDiffHash      func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

type commitmentvalidators struct {
	validateBlockNotNil                 func(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error
	validateTransactionsBlockMerkleRoot func(block *protocol.BlockPairContainer, vcx *validatorContext) error
	validateTransactionsMetadataHash    func(block *protocol.BlockPairContainer, vcx *validatorContext) error
	validateReceiptsMerkleRoot          func(block *protocol.BlockPairContainer, vcx *validatorContext) error
	validateResultsBlockStateDiffHash   func(block *protocol.BlockPairContainer, vcx *validatorContext) error
	validateBlockHash_Commitment        func(block *protocol.BlockPairContainer, vcx *validatorContext) error
}

func validateBlockNotNil(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error {
	if block == nil || block.TransactionsBlock == nil || block.ResultsBlock == nil {
		return errors.New("BlockPair or either transactions or results block are nil")
	}
	return nil
}

func validateTransactionsBlockMerkleRoot(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateTransactionsBlockMerkleRoot(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	})
}

func validateTransactionsMetadataHash(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateTransactionsBlockMetadataHash(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	})
}

func validateReceiptsMerkleRoot(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateReceiptsMerkleRoot(&validators.BlockValidatorContext{
		TransactionsBlock:      block.TransactionsBlock,
		ResultsBlock:           block.ResultsBlock,
		CalcReceiptsMerkleRoot: vcx.CalcReceiptsMerkleRoot,
	})
}

func validateResultsBlockStateDiffHash(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateResultsBlockStateDiffHash(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
		CalcStateDiffHash: vcx.CalcStateDiffHash,
	})
}

func validateBlockHash_Commitment(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateBlockHash(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
		ExpectedBlockHash: vcx.blockHash,
	})
}

func (p *blockProvider) ValidateBlockCommitment(blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash) bool {

	vcx := &validatorContext{
		blockHash:              primitives.Sha256(blockHash),
		CalcReceiptsMerkleRoot: digest.CalcReceiptsMerkleRoot,
		CalcStateDiffHash:      digest.CalcStateDiffHash,
	}
	return validateBlockCommitmentInternal(blockHeight, block, blockHash, p.logger, vcx, &commitmentvalidators{
		validateBlockNotNil:                 validateBlockNotNil,
		validateTransactionsBlockMerkleRoot: validateTransactionsBlockMerkleRoot,
		validateTransactionsMetadataHash:    validateTransactionsMetadataHash,
		validateReceiptsMerkleRoot:          validateReceiptsMerkleRoot,
		validateResultsBlockStateDiffHash:   validateResultsBlockStateDiffHash,
		validateBlockHash_Commitment:        validateBlockHash_Commitment,
	})
}

func validateBlockCommitmentInternal(blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash, logger log.BasicLogger, vcx *validatorContext, v *commitmentvalidators) bool {

	blockPair := FromLeanHelixBlock(block)

	validators := []blockValidator{
		v.validateBlockNotNil,
		v.validateTransactionsBlockMerkleRoot,
		v.validateTransactionsMetadataHash,
		v.validateReceiptsMerkleRoot,
		v.validateResultsBlockStateDiffHash,
		v.validateBlockHash_Commitment,
	}

	for _, validator := range validators {
		if err := validator(blockPair, vcx); err != nil {
			logger.Info("Error in ValidateBlockCommitment()", log.Error(err))
			return false
		}
	}

	return true
}
