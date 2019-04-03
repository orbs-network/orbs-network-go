package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type ethereumConnectorConfigForTests struct {
	endpoint                string
	privateKeyHex           string
	finalityTimeComponent   time.Duration
	finalityBlocksComponent uint32
}

func (c *ethereumConnectorConfigForTests) EthereumEndpoint() string {
	return c.endpoint
}

func (c *ethereumConnectorConfigForTests) EthereumFinalityTimeComponent() time.Duration {
	return c.finalityTimeComponent
}

func (c *ethereumConnectorConfigForTests) EthereumFinalityBlocksComponent() uint32 {
	return c.finalityBlocksComponent
}

func (c *ethereumConnectorConfigForTests) GetAuthFromConfig() (*bind.TransactOpts, error) {
	key, err := crypto.HexToECDSA(c.privateKeyHex)
	if err != nil {
		return nil, err
	}

	return bind.NewKeyedTransactor(key), nil
}

func TestReportingFailure(t *testing.T) {
	emptyConfig := &ethereumConnectorConfigForTests{}
	x := NewEthereumRpcConnection(emptyConfig, log.DefaultTestingLogger(t))
	err := x.updateConnectionStatus(context.Background(), createConnectionStatusMetrics(metric.NewRegistry()))
	require.Error(t, err, "require some error from the update flow, config is a lie")
}
