package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

type ethereumConnectorConfigForTests struct {
	endpoint string
}

func (c *ethereumConnectorConfigForTests) EthereumEndpoint() string {
	return c.endpoint
}

func TestEthBlockCacheInitAndRefresh(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	config := &ethereumConnectorConfigForTests{"https://mainnet.infura.io/v3/55322c8f5b9440f0940a37a3646eac76"} // using real endpoint
	conn := NewEthereumRpcConnection(config, logger)
	ctx := context.Background()

	// get the cache populated
	cache, err := conn.getBlocksCacheAndRefreshIfNeeded(ctx, time.Now().Unix()-200)
	require.NoError(t, err, "something went wrong refreshing cache from nil state")
	require.NotEqual(t, 0, cache.latest.number, "cache did not initialize correctly")
	require.NotEqual(t, 0, cache.latest.timestamp, "cache did not initialize correctly")
	require.NotEqual(t, 0, cache.back10k.number, "cache did not initialize correctly")
	require.NotEqual(t, 0, cache.back10k.timestamp, "cache did not initialize correctly")

	// mess up the data so it will refresh on next request
	cache.latest.timestamp = conn.mu.blocksCache.latest.timestamp - 10000
	messedUpCache := *cache

	// attempt cache refresh
	cache, err = conn.getBlocksCacheAndRefreshIfNeeded(ctx, time.Now().Unix()-200)
	require.NoError(t, err, "something went wrong refreshing cache from older state")
	require.True(t, messedUpCache.latest.timestamp < cache.latest.timestamp, "cache did not refresh correctly")
}

func TestEtcBlockCacheRetrieval(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	config := &ethereumConnectorConfigForTests{"https://mainnet.infura.io/v3/55322c8f5b9440f0940a37a3646eac76"} // using real endpoint
	conn := NewEthereumRpcConnection(config, logger)
	ctx := context.Background()

	// get the cache populated
	_, err := conn.getBlocksCacheAndRefreshIfNeeded(ctx, time.Now().Unix()-200)
	require.NoError(t, err, "something went wrong refreshing cache from nil state")

	start := time.Now()
	// attempt cache refresh (should not refresh
	_, err = conn.getBlocksCacheAndRefreshIfNeeded(ctx, time.Now().Unix()-200)
	require.NoError(t, err, "something went wrong refreshing cache from older state")
	require.True(t, time.Since(start).Nanoseconds()/int64(time.Millisecond) < 500, "cache failed, a refresh probably happened")
}
