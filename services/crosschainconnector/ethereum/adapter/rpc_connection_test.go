package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

type ethereumConnectorConfigForTests struct {
	endpoint string
}

func (c *ethereumConnectorConfigForTests) EthereumEndpoint() string {
	return c.endpoint
}

func TestEthBlockCacheInitAndRefresh(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	config := &ethereumConnectorConfigForTests{""} // using a fake backend, so no endpoint
	conn := NewEthereumRpcConnection(config, logger)
	conn.mu.fullClient = NewFakeFullClientConnection(logger)
	ctx := context.Background()

	// because this is using a fake backend, do not use 'time.now()' but use predefined values that match the fake database

	// get the cache populated
	cache, err := conn.getBlocksCacheAndRefreshIfNeeded(ctx, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED-200)
	require.NoError(t, err, "something went wrong refreshing cache from nil state")
	require.NotEqual(t, 0, cache.latest.number, "cache did not initialize correctly")
	require.NotEqual(t, 0, cache.latest.timestamp, "cache did not initialize correctly")
	require.NotEqual(t, 0, cache.back10k.number, "cache did not initialize correctly")
	require.NotEqual(t, 0, cache.back10k.timestamp, "cache did not initialize correctly")

	// mess up the data so it will refresh on next request
	cache.latest.timestamp = conn.mu.blocksCache.latest.timestamp - 10000
	messedUpCache := *cache

	// attempt cache refresh
	cache, err = conn.getBlocksCacheAndRefreshIfNeeded(ctx, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED-200)
	require.NoError(t, err, "something went wrong refreshing cache from older state")
	require.True(t, messedUpCache.latest.timestamp < cache.latest.timestamp, "cache did not refresh correctly")
	require.NotEqual(t, messedUpCache.lastUpdate, cache.lastUpdate, "cache did not refresh correctly")
}

func TestEtcBlockCacheRetrieval(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	config := &ethereumConnectorConfigForTests{""} // using a fake backend, no endpoint required
	conn := NewEthereumRpcConnection(config, logger)
	conn.mu.fullClient = NewFakeFullClientConnection(logger)
	ctx := context.Background()

	// get the cache populated
	original, err := conn.getBlocksCacheAndRefreshIfNeeded(ctx, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED-200)
	require.NoError(t, err, "something went wrong refreshing cache from nil state")

	// attempt cache refresh (should not refresh)
	noRefresh, err := conn.getBlocksCacheAndRefreshIfNeeded(ctx, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED-200)
	require.NoError(t, err, "something went wrong refreshing cache from older state")
	require.Equal(t, original.lastUpdate, noRefresh.lastUpdate, "cache failed, a refresh probably happened")
}

func TestGetEthBlockBeforeEthGenesis(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
		config := &ethereumConnectorConfigForTests{""} // no need for an endpoint at this test
		conn := NewEthereumRpcConnection(config, logger)
		// something before 2015/07/31
		_, err := conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1438300700000000000))
		require.Error(t, err, "expecting an error when trying to go too much into the past")
	})
}

func TestGetEthBlockByTimestampFromFutureFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
		config := &ethereumConnectorConfigForTests{""} // using a fake backend, so no endpoint
		conn := NewEthereumRpcConnection(config, logger)
		conn.mu.fullClient = NewFakeFullClientConnection(logger)

		// something in the future (sometime in 2031), it works on a fake database - which will never advance in time
		_, err := conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1944035343000000000))
		require.Error(t, err, "expecting an error when trying to go to the future")
	})
}

func TestGetEthBlockByTimestampFromEth(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
		config := &ethereumConnectorConfigForTests{""} // using a fake backend, so no endpoint
		conn := NewEthereumRpcConnection(config, logger)
		conn.mu.fullClient = NewFakeFullClientConnection(logger)

		// something recent
		blockBI, err := conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
		block := blockBI.Int64()
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a recent block")
		require.EqualValues(t, 938874, block, "expected ts 1505735343 to return a specific block")

		// something not so recent
		blockBI, err = conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1500198628000000000))
		block = blockBI.Int64()
		require.NoError(t, err, "something went wrong while getting the block by timestamp of an older block")
		require.EqualValues(t, 32599, block, "expected ts 1500198628 to return a specific block")

		// "realtime" - 200 seconds
		blockBI, err = conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1506108583000000000))
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a 'realtime' block")
		newBlock := blockBI.Int64()
		require.EqualValues(t, 999974, newBlock, "expected ts 1506108583 to return a specific block")
	})
}
