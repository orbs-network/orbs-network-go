package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
)

func getFinalitySafeBlockNumber(ctx context.Context, referenceTimestamp primitives.TimestampNano, timestampFetcher TimestampFetcher, config config.EthereumCrosschainConnectorConfig) (*big.Int, error) {
	// regard finality time component
	augmentedReferenceTimestamp := referenceTimestamp - primitives.TimestampNano(config.EthereumFinalityTimeComponent().Nanoseconds())

	// find the latest block number
	blockNumber, err := timestampFetcher.GetBlockByTimestamp(ctx, augmentedReferenceTimestamp)
	if err != nil {
		return nil, err
	}

	// geth simulator returns nil from GetBlockByTimestamp
	if blockNumber == nil {
		return nil, nil
	}

	// regard finality blocks component
	finalityBlocks := big.NewInt(int64(config.EthereumFinalityBlocksComponent()))
	blockAfterFinality := new(big.Int).Sub(blockNumber, finalityBlocks)

	if blockAfterFinality.Int64() < 0 {
		return nil, errors.Errorf("Got negative finality-safe block number %d; reference timestamp is %s, timestamp minus finality component is %s, block number correlating that time is %d", blockAfterFinality.Int64(), referenceTimestamp, augmentedReferenceTimestamp, blockNumber.Int64())
	}

	return blockAfterFinality, nil
}

func verifyBlockNumberIsFinalitySafe(ctx context.Context, blockNumber uint64, referenceTimestamp primitives.TimestampNano, timestampFetcher TimestampFetcher, config config.EthereumCrosschainConnectorConfig) error {
	safeBlockNumberBigInt, err := getFinalitySafeBlockNumber(ctx, referenceTimestamp, timestampFetcher, config)
	if err != nil {
		return err
	}

	// geth simulator returns nil from GetBlockByTimestamp
	if safeBlockNumberBigInt == nil {
		return nil
	}

	safeBlockNumber := safeBlockNumberBigInt.Uint64()
	if blockNumber > safeBlockNumber {
		return errors.Errorf("ethereum block number %d is unsafe for finality, latest safe block number is %d", blockNumber, safeBlockNumber)
	}

	return nil
}
