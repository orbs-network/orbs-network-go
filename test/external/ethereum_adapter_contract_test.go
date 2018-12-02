package external

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

type localconfig struct {
	endpoint      string
	privateKeyHex string
}

func (c *localconfig) EthereumEndpoint() string {
	return c.endpoint
}

func getConfig() *localconfig {
	var cfg localconfig

	if endpoint := os.Getenv("ETHEREUM_ENDPOINT"); endpoint != "" {
		cfg.endpoint = endpoint
	}

	if privateKey := os.Getenv("ETHEREUM_PRIVATE_KEY"); privateKey != "" {
		cfg.privateKeyHex = privateKey
	}

	return &cfg
}

//TODO refactor and make sense of: adapter directory, sdk_ethereum + its test
func TestEthereumNodeAdapter_Contract(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		address, adapter := createSimulatorAndDeploySimpleStorageContract(t)
		t.Run("Simulator Adapter", callSimpleStorageContractAndAssertReturnedValue(ctx, address, adapter))

		if os.Getenv("EXTERNAL_TEST") == "true" {
			address, adapter = connectViaRpcAndDeploySimpleStorageContract(t)
			t.Run("RPC Adapter", callSimpleStorageContractAndAssertReturnedValue(ctx, address, adapter))
		} else {
			t.Skip("skipping, external tests disabled")
		}
	})
}

func connectViaRpcAndDeploySimpleStorageContract(t *testing.T) ([]byte, adapter.EthereumConnection) {
	logger := log.GetLogger()
	cfg := getConfig()
	rpcClient := adapter.NewEthereumRpcConnection(cfg, logger)

	key, err := crypto.HexToECDSA(cfg.privateKeyHex)
	require.NoError(t, err, "failed generating key")
	auth := bind.NewKeyedTransactor(key)

	address, err := rpcClient.DeploySimpleStorageContract(auth, "foobar")
	require.NoError(t, err, "failed deploying contract")

	return address, rpcClient
}

func createSimulatorAndDeploySimpleStorageContract(t *testing.T) ([]byte, adapter.EthereumConnection) {
	logger := log.GetLogger()
	simulator := adapter.NewEthereumSimulatorConnection(logger)

	address, err := simulator.DeploySimpleStorageContract(simulator.GetAuth(), "foobar")
	simulator.Commit()
	require.NoError(t, err, "failed deploying contract to Ethereum")
	return address, simulator
}

func callSimpleStorageContractAndAssertReturnedValue(ctx context.Context, address []byte, connection adapter.EthereumConnection) func(t *testing.T) {
	return func(t *testing.T) {
		parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		require.NoError(t, err, "failed parsing ABI")

		packedInput, err := parsedABI.Pack("getString")
		require.NoError(t, err, "failed packing arguments")

		packedOutput, err := connection.CallContract(ctx, address, packedInput, nil)

		var out string
		err = parsedABI.Unpack(&out, "getString", packedOutput)
		require.NoError(t, err, "could not unpack call output")

		require.Equal(t, "foobar", out, "string output differed from expected")
	}
}
