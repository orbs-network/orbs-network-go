package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"math/big"
	"os"
	"strings"
)

type harness struct {
	adapter   adapter.EthereumConnection
	connector services.CrosschainConnector
	config    *ethereumConnectorConfigForTests
	logger    log.BasicLogger
	address   [20]byte
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

func (h *harness) withInvalidEndpoint() *harness {
	// mess up the config and use a real connector to see how it later behaves
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	config := newDefaultEthereumConnectorConfigForTests()
	conn := adapter.NewEthereumConnection(config, logger)
	ctx := context.Background()
	config.endpoint = "all your base"

	return &harness{
		adapter:   conn,
		connector: ethereum.NewEthereumCrosschainConnector(ctx, conn, logger),
	}
}

func (h *harness) deployStorageContract(ctx context.Context, number int64, text string) error {
	if err := h.adapter.Dial(""); err != nil { // create the client so we can deploy
		return err
	}
	client, err := h.adapter.GetClient()
	if err != nil {
		return err
	}
	address, _, _, err := DeploySimpleStorage(h.adapter.GetAuth(), client, big.NewInt(number), text)
	if err != nil {
		return err
	}
	h.address = address
	client.(*backends.SimulatedBackend).Commit() // assuming simulation, this will commit the pending transactions
	return nil
}

func (h *harness) getAddress() string {
	return hexutil.Encode(h.address[:])
}

func newEthereumConnectorHarness() *harness {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	config := newDefaultEthereumConnectorConfigForTests()
	conn := adapter.NewEthereumSimulatorConnection(config, logger)
	ctx := context.Background()

	return &harness{
		adapter:   conn,
		config:    config,
		logger:    logger,
		connector: ethereum.NewEthereumCrosschainConnector(ctx, conn, logger),
	}
}

func ethereumPackInputArguments(jsonAbi string, method string, args []interface{}) ([]byte, error) {
	if parsedABI, err := abi.JSON(strings.NewReader(jsonAbi)); err != nil {
		return nil, errors.WithStack(err)
	} else {
		return parsedABI.Pack(method, args...)
	}
}

func ethereumUnpackOutput(data []byte, method string, out interface{}) error {
	if parsedABI, err := abi.JSON(strings.NewReader(SimpleStorageABI)); err != nil {
		return errors.WithStack(err)
	} else {
		return parsedABI.Unpack(out, method, data)
	}
}
