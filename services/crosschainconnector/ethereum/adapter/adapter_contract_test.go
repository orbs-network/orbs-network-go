package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
)

const mnemonic = "vanish junk genuine web seminar cook absurd royal ability series taste method identify elevator liquid"
const privKeyHex = "f2ce3a9eddde6e5d996f6fe7c1882960b0e8ee8d799e0ef608276b8de4dc7f19"
const pubKeyHex = "037a809cc481303d337c1c83d1ba3a2222c7b1b820ac75e3c6f8dc63fa0ed79b18"
const dockerRun = "docker run -d -p 8545:8545 trufflesuite/ganache-cli:latest -a 10 -m \"vanish junk genuine web seminar cook absurd royal ability series taste method identify elevator liquid\""
const ethereumEndpoint = "http://localhost:8545"

type ethereumConnectorConfigForTests struct {
	endpoint string
}

func (c *ethereumConnectorConfigForTests) EthereumEndpoint() string {
	return c.endpoint
}

//TODO refactor and make sense of: adapter directory, sdk_ethereum + its test
func TestEthereumNodeAdapter_Contract(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		address, client := createSimulatorAndDeploySimpleStorageContract(ctx, t)
		t.Run("Simulator Adapter", callSimpleStorageContractAndAssertReturnedValue(ctx, address, client))

		address, client = connectViaRpcAndDeploySimpleStorageContract(t)
		t.Run("RPC Adapter", callSimpleStorageContractAndAssertReturnedValue(ctx, address, client))
	})
}

func connectViaRpcAndDeploySimpleStorageContract(t *testing.T) (*common.Address, bind.ContractBackend) {
	logger := log.GetLogger()
	cfg := &ethereumConnectorConfigForTests{endpoint: ethereumEndpoint}

	rpcClient := NewEthereumConnection(cfg, logger)

	key, err := crypto.HexToECDSA(privKeyHex)
	require.NoError(t, err, "failed generating key")
	auth := bind.NewKeyedTransactor(key)

	client, err := rpcClient.GetClient()
	require.NoError(t, err, "failed getting client")

	address, _, _, err := contract.DeploySimpleStorage(auth, client, big.NewInt(42), "foobar")
	require.NoError(t, err, "failed deploying contract")

	return &address, client
}

func createSimulatorAndDeploySimpleStorageContract(ctx context.Context, t *testing.T) (*common.Address, bind.ContractBackend) {
	logger := log.GetLogger()
	simulator := NewEthereumSimulatorConnection(logger)
	client, err := simulator.GetClient()
	require.NoError(t, err, "could not establish Ethereum connection")

	address, _, _, err := contract.DeploySimpleStorage(simulator.GetAuth(), client, big.NewInt(42), "foobar")
	simulator.Commit()
	require.NoError(t, err, "failed deploying contract to Ethereum")
	return &address, client
}

func callSimpleStorageContractAndAssertReturnedValue(ctx context.Context, address *common.Address, client bind.ContractBackend) func(t *testing.T) {
	return func(t *testing.T) {
		parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		require.NoError(t, err, "failed parsing ABI")

		packedInput, err := parsedABI.Pack("getString")
		require.NoError(t, err, "failed packing arguments")

		opts := new(bind.CallOpts)
		msg := ethereum.CallMsg{From: opts.From, To: address, Data: packedInput}
		packedOutput, err := client.CallContract(ctx, msg, nil)

		var out string
		err = parsedABI.Unpack(&out, "getString", packedOutput)
		require.NoError(t, err, "could not unpack call output")

		require.Equal(t, "foobar", out, "string output differed from expected")
	}
}