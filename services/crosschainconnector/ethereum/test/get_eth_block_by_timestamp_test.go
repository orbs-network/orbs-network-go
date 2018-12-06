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
)

func TestGetEthBlockByTimestampFromEth(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
		config := &ethereumConnectorConfigForTests{"https://mainnet.infura.io/v3/55322c8f5b9440f0940a37a3646eac76"} // using real endpoint
		conn := adapter.NewEthereumRpcConnection(config, logger)
		block, err := conn.GetBlockByTimestamp(ctx, primitives.TimestampNano(1544035343))
		require.NoError(t, err, "something went wrong while getting the block by timestamp")
		require.EqualValues(t, 6832127, block.Number, "expected ts 1544035343 to return a specific block")
	})
}
