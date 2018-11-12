package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"os"
)

type harness struct {
	adapter   adapter.EthereumConnection
	connector services.CrosschainConnector
}

type ethereumConnectorConfigForTests struct {
	endpoint string
}

func (c *ethereumConnectorConfigForTests) EthereumEndpoint() string {
	return c.endpoint
}

func newDefaultEthereumConnectorConfigForTests() *ethereumConnectorConfigForTests {
	return &ethereumConnectorConfigForTests{
		endpoint: "http://localhost:8545",
	}
}

func (h *harness) start() error {
	return nil
}

func newEthreumConnectorHarness() *harness {
	conn := adapter.NewEthereumSimulatorConnector()
	ctx := context.Background()
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))

	return &harness{
		adapter:   conn,
		connector: ethereum.NewEthereumCrosschainConnector(ctx, newDefaultEthereumConnectorConfigForTests(), conn, logger),
	}
}
