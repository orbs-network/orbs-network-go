package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/timestampfinder"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const RECENT_TIMESTAMP = primitives.TimestampNano(1505735343000000000)
const RECENT_BLOCK_NUMBER_OF_FAKE_GETTER = 938874

func TestFinality_GetSafeBlockWithoutLimits(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		cfg := &finalityConfig{0, 0}

		btg := timestampfinder.NewFakeBlockTimeGetter(logger)
		finder := timestampfinder.NewTimestampFinder(btg, logger)

		safeBlockNumber, err := getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP, finder, cfg)
		t.Log("safe block number is", safeBlockNumber)
		require.NoError(t, err, "should not fail")
		require.EqualValues(t, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER, safeBlockNumber.Uint64(), "should return the recent block number of fake getter")
	})
}

func TestFinality_GetSafeBlockWithBlockLimit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		cfg := &finalityConfig{0, 100}

		btg := timestampfinder.NewFakeBlockTimeGetter(logger)
		finder := timestampfinder.NewTimestampFinder(btg, logger)

		safeBlockNumber, err := getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP, finder, cfg)
		t.Log("safe block number is", safeBlockNumber)
		require.NoError(t, err, "should not fail")
		require.EqualValues(t, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-100, safeBlockNumber.Uint64(), "should return 100 blocks before the recent block number of fake getter")
	})
}

func TestFinality_GetSafeBlockWithTimeLimit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		cfg := &finalityConfig{200 * time.Second, 0}

		btg := timestampfinder.NewFakeBlockTimeGetter(logger)
		finder := timestampfinder.NewTimestampFinder(btg, logger)

		safeBlockNumber, err := getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP, finder, cfg)
		t.Log("safe block number is", safeBlockNumber)
		require.NoError(t, err, "should not fail")
		require.Truef(t, safeBlockNumber.Uint64() < RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-10, "should return at least 10 blocks before the recent block number of fake getter, but difference is %d", RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-safeBlockNumber.Uint64())
	})
}

func TestFinality_VerifySafeBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		cfg := &finalityConfig{0, 100}

		btg := timestampfinder.NewFakeBlockTimeGetter(logger)
		finder := timestampfinder.NewTimestampFinder(btg, logger)

		err := verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-100, RECENT_TIMESTAMP, finder, cfg)
		require.NoError(t, err, "100 difference should be safe")

		err = verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-101, RECENT_TIMESTAMP, finder, cfg)
		require.NoError(t, err, "101 difference should be safe")

		err = verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-99, RECENT_TIMESTAMP, finder, cfg)
		require.Error(t, err, "99 difference should not be safe")
	})
}

type finalityConfig struct {
	finalityTimeComponent   time.Duration
	finalityBlocksComponent uint32
}

func (c *finalityConfig) EthereumFinalityTimeComponent() time.Duration {
	return c.finalityTimeComponent
}

func (c *finalityConfig) EthereumFinalityBlocksComponent() uint32 {
	return c.finalityBlocksComponent
}
