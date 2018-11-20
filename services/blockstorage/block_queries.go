package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/bloom"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) GetLastCommittedBlockHeight(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
	b, err := s.persistence.GetLastBlock()
	if err != nil {
		return nil, err
	}
	return &services.GetLastCommittedBlockHeightOutput{
		LastCommittedBlockHeight:    getBlockHeight(b),
		LastCommittedBlockTimestamp: getBlockTimestamp(b),
	}, nil
}

func (s *service) loadTransactionsBlockHeader(height primitives.BlockHeight) (*services.GetTransactionsBlockHeaderOutput, error) {
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

func (s *service) GetTransactionsBlockHeader(ctx context.Context, input *services.GetTransactionsBlockHeaderInput) (result *services.GetTransactionsBlockHeaderOutput, err error) {
	err = s.persistence.GetBlockTracker().WaitForBlock(ctx, input.BlockHeight)
	if err == nil {
		return s.loadTransactionsBlockHeader(input.BlockHeight)
	}
	return nil, err
}

func (s *service) loadResultsBlockHeader(height primitives.BlockHeight) (*services.GetResultsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetResultsBlock(height)
	if err != nil {
		return nil, err
	}
	return &services.GetResultsBlockHeaderOutput{
		ResultsBlockProof:  txBlock.BlockProof,
		ResultsBlockHeader: txBlock.Header,
	}, nil
}

func (s *service) GetResultsBlockHeader(ctx context.Context, input *services.GetResultsBlockHeaderInput) (result *services.GetResultsBlockHeaderOutput, err error) {
	err = s.persistence.GetBlockTracker().WaitForBlock(ctx, input.BlockHeight)
	if err == nil {
		return s.loadResultsBlockHeader(input.BlockHeight)
	}
	return nil, err
}

// TODO: are we sure that if we don't find the receipt this API should fail? it should succeed just return nil receipt
func (s *service) GetTransactionReceipt(ctx context.Context, input *services.GetTransactionReceiptInput) (*services.GetTransactionReceiptOutput, error) {
	searchRules := adapter.BlockSearchRules{
		EndGraceNano:          s.config.BlockTransactionReceiptQueryGraceEnd().Nanoseconds(),
		StartGraceNano:        s.config.BlockTransactionReceiptQueryGraceStart().Nanoseconds(),
		TransactionExpireNano: s.config.BlockTransactionReceiptQueryExpirationWindow().Nanoseconds(),
	}
	blocksToSearch := s.persistence.GetBlocksRelevantToTxTimestamp(input.TransactionTimestamp, searchRules)
	if blocksToSearch == nil {
		receipt, err := s.createEmptyTransactionReceiptResult(ctx)
		if err != nil {
			return nil, err
		}
		// TODO: probably don't fail here (issue#448)
		return receipt, errors.Errorf("failed to search for blocks on tx timestamp of %d, hash %s", input.TransactionTimestamp, input.Txhash)
	}

	if len(blocksToSearch) == 0 {
		// duplication of this piece of code is a smell originating from issue#448
		receipt, err := s.createEmptyTransactionReceiptResult(ctx)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	}

	for _, b := range blocksToSearch {
		tbf := bloom.NewFromRaw(b.ResultsBlock.Header.TimestampBloomFilter())
		if tbf.Test(input.TransactionTimestamp) {
			for _, txr := range b.ResultsBlock.TransactionReceipts {
				if txr.Txhash().Equal(input.Txhash) {
					return &services.GetTransactionReceiptOutput{
						TransactionReceipt: txr,
						BlockHeight:        b.ResultsBlock.Header.BlockHeight(),
						BlockTimestamp:     b.ResultsBlock.Header.Timestamp(),
					}, nil
				}
			}
		}
	}

	receipt, err := s.createEmptyTransactionReceiptResult(ctx)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// Returns a slice of blocks containing first and last
// TODO support paging
func (s *service) GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight, err error) {
	return s.persistence.GetBlocks(first, last)
}

func (s *service) createEmptyTransactionReceiptResult(ctx context.Context) (*services.GetTransactionReceiptOutput, error) {
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

