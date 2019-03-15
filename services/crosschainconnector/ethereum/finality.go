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
	return new(big.Int).Sub(blockNumber, finalityBlocks), nil
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
