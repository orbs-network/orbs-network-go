package leanhelixconsensus

import (
	"context"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type validateBlockProposalContext struct {
	logger                    log.BasicLogger
	validateTransactionsBlock func(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error)
	validateResultsBlock      func(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error)
	validateBlockHash         func(blockHash primitives.Sha256, tx *protocol.TransactionsBlockContainer, rx *protocol.ResultsBlockContainer) error
}

// Block height is unused - the spec of ValidateBlockProposal() prepares for a height-based config but it is not part of v1
func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash, prevBlock lh.Block) bool {
	return validateBlockProposalInternal(ctx, block, blockHash, prevBlock, &validateBlockProposalContext{
		validateTransactionsBlock: p.consensusContext.ValidateTransactionsBlock,
		validateResultsBlock:      p.consensusContext.ValidateResultsBlock,
		validateBlockHash:         validateBlockHash_Proposal,
		logger:                    p.logger,
	})
}

func validateBlockProposalInternal(ctx context.Context, block lh.Block, blockHash lhprimitives.BlockHash, prevBlock lh.Block, vctx *validateBlockProposalContext) bool {
	blockPair := FromLeanHelixBlock(block)

	if blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.ResultsBlock == nil {
		vctx.logger.Info("Error in ValidateBlockProposal(): block or its tx/rx are nil")
		return false
	}

	newBlockHeight := primitives.BlockHeight(1)
	var prevTxBlockHash primitives.Sha256
	var prevRxBlockHash primitives.Sha256
	//var prevBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) - 1
	var prevBlockTimestamp primitives.TimestampNano

	if prevBlock != nil {
		prevBlockPair := FromLeanHelixBlock(prevBlock)
		newBlockHeight = primitives.BlockHeight(prevBlock.Height() + 1)
		prevTxBlock := prevBlockPair.TransactionsBlock
		prevTxBlockHash = digest.CalcTransactionsBlockHash(prevTxBlock)
		prevBlockTimestamp = prevTxBlock.Header.Timestamp()
		prevRxBlockHash = digest.CalcResultsBlockHash(prevBlockPair.ResultsBlock)
	}

	_, err := vctx.validateTransactionsBlock(ctx, &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight: newBlockHeight,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockHash:      prevTxBlockHash,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		vctx.logger.Error("ValidateBlockProposal failed ValidateTransactionsBlock", log.Error(err), log.BlockHeight(newBlockHeight))
		return false
	}

	_, err = vctx.validateResultsBlock(ctx, &services.ValidateResultsBlockInput{
		CurrentBlockHeight: newBlockHeight,
		ResultsBlock:       blockPair.ResultsBlock,
		PrevBlockHash:      prevRxBlockHash,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		vctx.logger.Error("ValidateBlockProposal failed ValidateResultsBlock", log.BlockHeight(newBlockHeight), log.Error(err))
		return false
	}

	err = vctx.validateBlockHash(primitives.Sha256(blockHash), blockPair.TransactionsBlock, blockPair.ResultsBlock)
	if err != nil {
		vctx.logger.Error("ValidateBlockProposal blockHash mismatch", log.Error(err), log.Stringable("expected-block-hash", blockHash))
		return false
	}
	vctx.logger.Info("ValidateBlockProposal passed", log.BlockHeight(newBlockHeight))
	return true
}

func validateBlockHash_Proposal(blockHash primitives.Sha256, tx *protocol.TransactionsBlockContainer, rx *protocol.ResultsBlockContainer) error {
	return validators.ValidateBlockHash(&validators.BlockValidatorContext{
		TransactionsBlock: tx,
		ResultsBlock:      rx,
		ExpectedBlockHash: blockHash,
	})
}
