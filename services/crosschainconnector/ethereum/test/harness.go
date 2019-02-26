package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

type harness struct {
	simAdapter *adapter.EthereumSimulator
	rpcAdapter adapter.DeployingEthereumConnection
	connector  services.CrosschainConnector
	logger     log.BasicLogger
	address    string
	config     *ethereumConnectorConfigForTests
}

type ethereumConnectorConfigForTests struct {
	endpoint      string
	privateKeyHex string
}

func (c *ethereumConnectorConfigForTests) EthereumEndpoint() string {
	return c.endpoint
}

func (h *harness) deploySimulatorStorageContract(ctx context.Context, text string) error {
	address, err := h.simAdapter.DeploySimpleStorageContract(h.simAdapter.GetAuth(), text)
	h.simAdapter.Commit()
	if err != nil {
		return err
	}

	h.address = hexutil.Encode(address[:])
	return nil
}

func (h *harness) getAddress() string {
	return h.address
}

func (h *harness) deployRpcStorageContract(text string) (string, error) {
	auth, err := h.authFromConfig()
	if err != nil {
		return "", err
	}
	address, err := h.rpcAdapter.DeploySimpleStorageContract(auth, text)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(address[:]), nil
}

func (h *harness) deployContractsToGanache(t *testing.T, count int, delayBetweenContracts time.Duration) error {
	// create two blocks, in ganache transaction -> block
	for i := 0; i < count; i++ {
		_, err := h.deployRpcStorageContract("junk-we-do-not-care-about")
		require.NoError(t, err, "failed deploying contract number %d to Ethereum", i)

		time.Sleep(delayBetweenContracts)
	}

	return nil
}

func newRpcEthereumConnectorHarness(tb testing.TB, cfg *ethereumConnectorConfigForTests) *harness {
	logger := log.DefaultTestingLogger(tb)
	a := adapter.NewEthereumRpcConnection(cfg, logger)

	return &harness{
		config:     cfg,
		rpcAdapter: a,
		logger:     logger,
		connector:  ethereum.NewEthereumCrosschainConnector(a, logger),
	}
}

func (h *harness) authFromConfig() (*bind.TransactOpts, error) {
	key, err := crypto.HexToECDSA(h.config.privateKeyHex)
	if err != nil {
		return nil, err
	}

	return bind.NewKeyedTransactor(key), nil
}

func (h *harness) WithFakeTSF() *harness {
	h.connector = ethereum.NewEthereumCrosschainConnectorWithFakeTSF(h.simAdapter, h.logger)
	return h
}

func newSimulatedEthereumConnectorHarness(tb testing.TB) *harness {
	logger := log.DefaultTestingLogger(tb)
	conn := adapter.NewEthereumSimulatorConnection(logger)

	return &harness{
		simAdapter: conn,
		logger:     logger,
		connector:  ethereum.NewEthereumCrosschainConnector(conn, logger),
	}
}

func (h *harness) packInputArgumentsForSampleStorage(method string, args []interface{}) ([]byte, error) {
	if parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI)); err != nil {
		return nil, errors.WithStack(err)
	} else {
		return ethereum.ABIPackFunctionInputArguments(parsedABI, method, args)
	}
}
