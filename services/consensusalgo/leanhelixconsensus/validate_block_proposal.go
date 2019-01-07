package leanhelixconsensus

import (
	"bytes"
	"context"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"

	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash, prevBlock lh.Block) bool {

	// Validate Not Nil
	blockPair := FromLeanHelixBlock(block)

	validators := []blockValidator{
		validateBlockNotNil,
	}

	vcx := &validatorContext{}

	for _, validator := range validators {
		if err := validator(blockPair, vcx); err != nil {
			p.logger.Info("Error in ValidateBlockProposal()", log.Error(err))
			return false
		}
	}

	newBlockHeight := primitives.BlockHeight(1)
	var prevTxBlockHash primitives.Sha256 = nil
	var prevRxBlockHash primitives.Sha256 = nil
	//var prevBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) - 1
	var prevBlockTimestamp primitives.TimestampNano = 0

	if prevBlock != nil {
		prevBlockPair := FromLeanHelixBlock(prevBlock)
		newBlockHeight = primitives.BlockHeight(prevBlock.Height() + 1)
		prevTxBlock := prevBlockPair.TransactionsBlock
		prevTxBlockHash = digest.CalcTransactionsBlockHash(prevTxBlock)
		prevBlockTimestamp = prevTxBlock.Header.Timestamp()
		prevRxBlockHash = digest.CalcResultsBlockHash(prevBlockPair.ResultsBlock)
	}

	_, err := p.consensusContext.ValidateTransactionsBlock(ctx, &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight: newBlockHeight,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockHash:      prevTxBlockHash,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		p.logger.Error("ValidateBlockProposal failed ValidateTransactionsBlock", log.Error(err))
		return false
	}

	_, err = p.consensusContext.ValidateResultsBlock(ctx, &services.ValidateResultsBlockInput{
		CurrentBlockHeight: newBlockHeight,
		ResultsBlock:       blockPair.ResultsBlock,
		PrevBlockHash:      prevRxBlockHash,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		p.logger.Error("ValidateBlockProposal failed ValidateResultsBlock", log.Int("block-height", int(newBlockHeight)), log.Error(err))
		return false
	}

	calcBlockHash := []byte(digest.CalcBlockHash(blockPair.TransactionsBlock, blockPair.ResultsBlock))
	if !bytes.Equal([]byte(blockHash), calcBlockHash) {
		p.logger.Error("ValidateBlockProposal blockHash mismatch")
		return false
	}
	p.logger.Info("ValidateBlockProposal passed", log.Int("block-height", int(newBlockHeight)))
	return true
}

// TODO Remove this reference code when done digesting it
/*

// Called to  validate leader block proposal.
// Full block validation - content (block_height, prevBlock pointer, timeStamp, protocolVersion, virtualChainId - using consensusContext)
// and structure (inner hashPointers and blockHash).
// Assume prevBlock is valid and under consensus.
// Assume consensusContext validation also checks inner hashPointers
func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, block leanhelix.Block, blockHash lhprimitives.BlockHash, prevBlock leanhelix.Block) bool {
	blockPair := fromLeanHelixBlock(block)
	if blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.ResultsBlock == nil {
		p.logger.Error("ValidateBlockProposal received an invalid block containing nil")
		return false
	}

	newBlockHeight := primitives.BlockHeight(1)
	var prevTxBlockHash primitives.Sha256 = nil
	var prevRxBlockHash primitives.Sha256 = nil
	//var prevBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) - 1
	var prevBlockTimestamp primitives.TimestampNano = 0

	if prevBlock != nil {
		prevBlockPair := fromLeanHelixBlock(prevBlock)
		newBlockHeight = primitives.BlockHeight(prevBlock.Height() + 1)
		prevTxBlock := prevBlockPair.TransactionsBlock
		prevTxBlockHash = digest.CalcTransactionsBlockHash(prevTxBlock)
		prevBlockTimestamp = prevTxBlock.Header.Timestamp()
		prevRxBlockHash = digest.CalcResultsBlockHash(prevBlockPair.ResultsBlock)
	}

	// also inner hashPointers are checked
	_, err := p.consensusContext.ValidateTransactionsBlock(ctx, &services.ValidateTransactionsBlockInput{
		BlockHeight:        newBlockHeight,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockHash:      prevTxBlockHash,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		p.logger.Error("ValidateBlockProposal failed ValidateTransactionsBlock", log.Error(err))
		return false
	}

	// also inner hashPointers are checked
	_, err = p.consensusContext.ValidateResultsBlock(ctx, &services.ValidateResultsBlockInput{
		BlockHeight:        newBlockHeight,
		ResultsBlock:       blockPair.ResultsBlock,
		PrevBlockHash:      prevRxBlockHash,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		p.logger.Error("ValidateBlockProposal failed ValidateResultsBlock", log.Int("block-height", int(newBlockHeight)), log.Error(err))
		return false
	}

	// validate blockHash
	calcBlockHash := []byte(digest.CalcBlockHash(blockPair.TransactionsBlock, blockPair.ResultsBlock))
	if !bytes.Equal([]byte(blockHash), calcBlockHash) {
		p.logger.Error("ValidateBlockProposal blockHash mismatch")
		return false
	}
	p.logger.Info("ValidateBlockProposal passed", log.Int("block-height", int(newBlockHeight)))
	return true
}

*/
