package leanhelixconsensus

import (
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"

	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type blockValidator func(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error

type validatorContext struct {
	blockHash primitives.Sha256
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
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	})
}

func validateResultsBlockStateDiffHash(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateResultsBlockStateDiffHash(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	})
}

func validateBlockHash(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateBlockHash(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
		ExpectedBlockHash: vcx.blockHash,
	})
}

func (p *blockProvider) ValidateBlockCommitment(blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash) bool {

	blockPair := FromLeanHelixBlock(block)

	validators := []blockValidator{
		validateBlockNotNil,
		validateTransactionsBlockMerkleRoot,
		validateTransactionsMetadataHash,
		validateReceiptsMerkleRoot,
		validateResultsBlockStateDiffHash,
		validateBlockHash,
	}

	vcx := &validatorContext{
		blockHash: primitives.Sha256(blockHash),
	}

	for _, validator := range validators {
		if err := validator(blockPair, vcx); err != nil {
			p.logger.Info("Error in ValidateBlockCommitment()")
			return false
		}
	}

	return true
}
