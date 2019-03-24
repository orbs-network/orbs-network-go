// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/timestampfinder"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
)

func getFinalitySafeBlockNumber(ctx context.Context, referenceTimestamp primitives.TimestampNano, timestampFinder timestampfinder.TimestampFinder, config config.EthereumCrosschainConnectorConfig) (*big.Int, error) {
	// regard finality time component
	augmentedReferenceTimestamp := referenceTimestamp - primitives.TimestampNano(config.EthereumFinalityTimeComponent().Nanoseconds())

	// find the latest block number
	blockNumber, err := timestampFinder.FindBlockByTimestamp(ctx, augmentedReferenceTimestamp)
	if err != nil {
		return nil, err
	}

	// geth simulator returns nil from FindBlockByTimestamp
	if blockNumber == nil {
		return nil, nil
	}

	// regard finality blocks component
	finalityBlocks := big.NewInt(int64(config.EthereumFinalityBlocksComponent()))
	result := new(big.Int).Sub(blockNumber, finalityBlocks)

	// make sure result is not below 1 (the first block)
	if result.Cmp(big.NewInt(1)) < 0 {
		return nil, errors.Errorf("there are not enough blocks to reach a finality safe block, finality safe block is %v", result)
	}

	return result, nil
}

func verifyBlockNumberIsFinalitySafe(ctx context.Context, blockNumber uint64, referenceTimestamp primitives.TimestampNano, timestampFinder timestampfinder.TimestampFinder, config config.EthereumCrosschainConnectorConfig) error {
	safeBlockNumberBigInt, err := getFinalitySafeBlockNumber(ctx, referenceTimestamp, timestampFinder, config)
	if err != nil {
		return err
	}

	// geth simulator returns nil from FindBlockByTimestamp
	if safeBlockNumberBigInt == nil {
		return nil
	}

	safeBlockNumber := safeBlockNumberBigInt.Uint64()
	if blockNumber > safeBlockNumber {
		return errors.Errorf("ethereum block number %d is unsafe for finality, latest safe block number is %d", blockNumber, safeBlockNumber)
	}

	return nil
}
