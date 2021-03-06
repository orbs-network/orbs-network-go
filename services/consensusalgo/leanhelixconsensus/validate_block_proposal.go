// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/crypto-lib-go/crypto/validators"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

type validateBlockProposalContext struct {
	logger                    log.Logger
	validateTransactionsBlock func(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error)
	validateResultsBlock      func(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error)
	validateBlockHash         func(blockHash primitives.Sha256, tx *protocol.TransactionsBlockContainer, rx *protocol.ResultsBlockContainer) error
}

// Block height is unused - the spec of ValidateBlockProposal() prepares for a height-based config but it is not part of v1
func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, blockProposer lhprimitives.MemberId, block lh.Block, blockHash lhprimitives.BlockHash, prevBlock lh.Block) error {
	return validateBlockProposalInternal(ctx, block, blockHash, blockProposer, prevBlock, &validateBlockProposalContext{
		validateTransactionsBlock: p.consensusContext.ValidateTransactionsBlock,
		validateResultsBlock:      p.consensusContext.ValidateResultsBlock,
		validateBlockHash:         validateBlockHash_Proposal,
		logger:                    p.logger,
	})
}

func validateBlockProposalInternal(ctx context.Context, block lh.Block, blockHash lhprimitives.BlockHash, blockProposer lhprimitives.MemberId, prevBlock lh.Block, vctx *validateBlockProposalContext) error {
	blockPair := FromLeanHelixBlock(block)

	if blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.ResultsBlock == nil {
		return errors.New("block or its tx/rx are nil")
	}

	newBlockHeight := primitives.BlockHeight(1)
	blockProposerAddress := primitives.NodeAddress(blockProposer) // TODO Noam maybe need an empty convertor in lhprimitives

	var prevTxBlockHash primitives.Sha256
	var prevRxBlockHash primitives.Sha256
	//var prevBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) - 1
	var prevBlockTimestamp primitives.TimestampNano
	var prevBlockReferenceTime primitives.TimestampSeconds

	if prevBlock != nil {
		prevBlockPair := FromLeanHelixBlock(prevBlock)
		newBlockHeight = primitives.BlockHeight(prevBlock.Height() + 1)
		prevTxBlock := prevBlockPair.TransactionsBlock
		prevTxBlockHash = digest.CalcTransactionsBlockHash(prevTxBlock)
		prevBlockTimestamp = prevTxBlock.Header.Timestamp()
		prevRxBlockHash = digest.CalcResultsBlockHash(prevBlockPair.ResultsBlock)
		prevBlockReferenceTime = prevTxBlock.Header.ReferenceTime()
	}

	_, err := vctx.validateTransactionsBlock(ctx, &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight:     newBlockHeight,
		TransactionsBlock:      blockPair.TransactionsBlock,
		PrevBlockHash:          prevTxBlockHash,
		PrevBlockTimestamp:     prevBlockTimestamp,
		PrevBlockReferenceTime: prevBlockReferenceTime,
		BlockProposerAddress:   blockProposerAddress,
	})
	if err != nil {
		return errors.Wrapf(err, "ValidateBlockProposal failed ValidateTransactionsBlock, newBlockHeight=%d", newBlockHeight)
	}

	_, err = vctx.validateResultsBlock(ctx, &services.ValidateResultsBlockInput{
		CurrentBlockHeight:     newBlockHeight,
		ResultsBlock:           blockPair.ResultsBlock,
		PrevBlockHash:          prevRxBlockHash,
		TransactionsBlock:      blockPair.TransactionsBlock,
		PrevBlockTimestamp:     prevBlockTimestamp,
		PrevBlockReferenceTime: prevBlockReferenceTime,
		BlockProposerAddress:   blockProposerAddress,
	})
	if err != nil {
		return errors.Wrapf(err, "ValidateBlockProposal failed ValidateResultsBlock, newBlockHeight=%d", newBlockHeight)
	}

	err = vctx.validateBlockHash(primitives.Sha256(blockHash), blockPair.TransactionsBlock, blockPair.ResultsBlock)
	if err != nil {
		return errors.Wrapf(err, "ValidateBlockProposal blockHash mismatch, expectedBlockHash=%s", blockHash)
	}
	vctx.logger.Info("ValidateBlockProposal passed", logfields.BlockHeight(newBlockHeight))
	return nil
}

func validateBlockHash_Proposal(blockHash primitives.Sha256, tx *protocol.TransactionsBlockContainer, rx *protocol.ResultsBlockContainer) error {
	return validators.ValidateBlockHash(&validators.BlockValidatorContext{
		TransactionsBlock: tx,
		ResultsBlock:      rx,
		ExpectedBlockHash: blockHash,
	})
}
