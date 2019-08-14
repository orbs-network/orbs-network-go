// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *Service) GetLastCommittedBlockHeight(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
	b, err := s.persistence.GetLastBlock()
	if err != nil {
		return nil, err
	}
	return &services.GetLastCommittedBlockHeightOutput{
		LastCommittedBlockHeight:    getBlockHeight(b),
		LastCommittedBlockTimestamp: getBlockTimestamp(b),
	}, nil
}

func (s *Service) loadTransactionsBlockHeader(height primitives.BlockHeight) (*services.GetTransactionsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetTransactionsBlock(height)
	if err != nil {
		return nil, err
	}
	return &services.GetTransactionsBlockHeaderOutput{
		TransactionsBlockProof:    txBlock.BlockProof,
		TransactionsBlockHeader:   txBlock.Header,
		TransactionsBlockMetadata: txBlock.Metadata,
	}, nil
}

func (s *Service) GetTransactionsBlockHeader(ctx context.Context, input *services.GetTransactionsBlockHeaderInput) (result *services.GetTransactionsBlockHeaderOutput, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	defer cancel()

	err = s.persistence.GetBlockTracker().WaitForBlock(timeoutCtx, input.BlockHeight)
	if err == nil {
		return s.loadTransactionsBlockHeader(input.BlockHeight)
	}
	return nil, err
}

func (s *Service) loadResultsBlockHeader(height primitives.BlockHeight) (*services.GetResultsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetResultsBlock(height)
	if err != nil {
		return nil, err
	}
	return &services.GetResultsBlockHeaderOutput{
		ResultsBlockProof:  txBlock.BlockProof,
		ResultsBlockHeader: txBlock.Header,
	}, nil
}

func (s *Service) GetResultsBlockHeader(ctx context.Context, input *services.GetResultsBlockHeaderInput) (result *services.GetResultsBlockHeaderOutput, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	defer cancel()

	err = s.persistence.GetBlockTracker().WaitForBlock(timeoutCtx, input.BlockHeight)
	if err == nil {
		return s.loadResultsBlockHeader(input.BlockHeight)
	}
	return nil, err
}

func (s *Service) GetTransactionReceipt(ctx context.Context, input *services.GetTransactionReceiptInput) (*services.GetTransactionReceiptOutput, error) {
	graceNano := s.config.BlockStorageTransactionReceiptQueryTimestampGrace().Nanoseconds()
	txExpireNano := s.config.TransactionExpirationWindow().Nanoseconds()

	start := input.TransactionTimestamp - primitives.TimestampNano(graceNano)
	end := input.TransactionTimestamp + primitives.TimestampNano(graceNano+txExpireNano)

	blockPair, txIdx, err := s.persistence.GetBlockByTx(input.Txhash, start, end)
	if err != nil {
		return nil, err
	}
	if blockPair == nil {
		receipt, err := s.createEmptyTransactionReceiptResult(ctx)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	}

	return &services.GetTransactionReceiptOutput{
		TransactionReceipt: blockPair.ResultsBlock.TransactionReceipts[txIdx],
		BlockHeight:        blockPair.ResultsBlock.Header.BlockHeight(),
		BlockTimestamp:     blockPair.ResultsBlock.Header.Timestamp(),
	}, nil
}

// Returns a slice of blocks containing first and last
// TODO kill this method signature or use a larger page size without returning too many blocks
func (s *Service) GetBlockSlice(first primitives.BlockHeight, last primitives.BlockHeight) ([]*protocol.BlockPairContainer, primitives.BlockHeight, primitives.BlockHeight, error) {
	blocks := make([]*protocol.BlockPairContainer, 0, last-first+1)
	err := s.persistence.ScanBlocks(first, 1, func(firstInPage primitives.BlockHeight, page []*protocol.BlockPairContainer) bool {
		blocks = append(blocks, page...)
		return firstInPage < last
	})
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "failed getting block slice")
	}
	if len(blocks) == 0 {
		return nil, 0, 0, fmt.Errorf("could not find blocks in height range %d-%d", first, last)
	}
	return blocks, first, first + primitives.BlockHeight(len(blocks)) - 1, nil
}

func (s *Service) createEmptyTransactionReceiptResult(ctx context.Context) (*services.GetTransactionReceiptOutput, error) {
	out, err := s.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return nil, err
	}
	return &services.GetTransactionReceiptOutput{
		TransactionReceipt: nil,
		BlockHeight:        out.LastCommittedBlockHeight,
		BlockTimestamp:     out.LastCommittedBlockTimestamp,
	}, nil
}

func (s *Service) GetBlockPair(ctx context.Context, input *services.GetBlockPairInput) (*services.GetBlockPairOutput, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.BlockTrackerGraceTimeout())
	defer cancel()

	err := s.persistence.GetBlockTracker().WaitForBlock(timeoutCtx, input.BlockHeight)
	if err != nil {
		return &services.GetBlockPairOutput{
			BlockPair: nil,
		}, nil
	}

	var bpc *protocol.BlockPairContainer
	err = s.persistence.ScanBlocks(input.BlockHeight, 1, func(h primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
		bpc = page[0]
		return false
	})
	if err != nil {
		return nil, err
	}

	return &services.GetBlockPairOutput{
		BlockPair: bpc,
	}, nil
}
