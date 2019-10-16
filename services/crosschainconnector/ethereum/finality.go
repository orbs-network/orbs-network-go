// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/timestampfinder"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
)

func (s *service) getFinalitySafeBlockNumber(ctx context.Context, referenceTimestamp primitives.TimestampNano) (*timestampfinder.BlockNumberAndTime, error) {
	// regard finality time component
	augmentedReferenceTimestamp := referenceTimestamp - primitives.TimestampNano(s.config.EthereumFinalityTimeComponent().Nanoseconds())

	// find the latest block number
	blockNumberAndTime, err := s.timestampFinder.FindBlockByTimestamp(ctx, augmentedReferenceTimestamp)
	if err != nil {
		return nil, err
	}

	// regard finality blocks component
	finalityBlocks := int64(s.config.EthereumFinalityBlocksComponent())
	resultBlock := blockNumberAndTime.BlockNumber - finalityBlocks

	// make sure result is not below 1 (the first block)
	if resultBlock < 1 {
		return nil, errors.Errorf("there are not enough blocks to reach a finality safe block, finality safe block is %v", resultBlock)
	}

	result, err := s.blockTimeGetter.GetTimestampForBlockNumber(ctx, big.NewInt(resultBlock))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *service) verifyBlockNumberIsFinalitySafe(ctx context.Context, blockNumber uint64, referenceTimestamp primitives.TimestampNano) error {
	safeBlockNumber, err := s.getFinalitySafeBlockNumber(ctx, referenceTimestamp)
	if err != nil {
		return err
	}

	if blockNumber > uint64(safeBlockNumber.BlockNumber) {
		return errors.Errorf("ethereum block number %d is unsafe for finality, latest safe block number is %d", blockNumber, safeBlockNumber)
	}

	return nil
}
