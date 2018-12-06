package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestGetEthBlockByTimestampFromFutureFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
		config := &ethereumConnectorConfigForTests{"https://mainnet.infura.io/v3/55322c8f5b9440f0940a37a3646eac76"} // using real endpoint
		conn := adapter.NewEthereumRpcConnection(config, logger)
		// something recent
		_, err := conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1944035343000000000))
		require.Error(t, err, "expecting an error when trying to go to the future")
	})
}

func TestGetEthBlockByTimestampFromEth(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
		config := &ethereumConnectorConfigForTests{"https://mainnet.infura.io/v3/55322c8f5b9440f0940a37a3646eac76"} // using real endpoint
		conn := adapter.NewEthereumRpcConnection(config, logger)
		// something recent
		block, err := conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1544035343000000000))
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a recent block")
		require.EqualValues(t, 6832126, block, "expected ts 1544035343 to return a specific block")

		// something not so recent
		block, err = conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1532168628000000000))
		require.NoError(t, err, "something went wrong while getting the block by timestamp of an older block")
		require.EqualValues(t, 6003358, block, "expected ts 1532168628 to return a specific block")

		// "realtime" - 200 seconds
		backTwoHundredsSeconds := time.Now().UnixNano() - (int64(time.Second) * 200)
		_, err = conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(backTwoHundredsSeconds))
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a 'realtime' block")
	})
}
