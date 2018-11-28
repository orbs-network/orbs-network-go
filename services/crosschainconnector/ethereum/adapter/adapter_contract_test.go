package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
)

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

		address, client = connectViaRpcAndDeploySimpleStorageContract(ctx, t)
		t.Run("RPC Adapter", callSimpleStorageContractAndAssertReturnedValue(ctx, address, client))
	})
}

func connectViaRpcAndDeploySimpleStorageContract(ctx context.Context, t *testing.T) (string, bind.ContractBackend) {
	logger := log.GetLogger()
	cfg := &ethereumConnectorConfigForTests{endpoint: "http://localhost:8545"}

	rpcClient := NewEthereumConnection(cfg, logger)

	key, err := crypto.HexToECDSA("97e30dc07275eae3359b3abd87d81ffe295255914276a21d95e5fe5a0bee610b")
	require.NoError(t, err, "failed generating key")
	auth := bind.NewKeyedTransactor(key)

	client, err := rpcClient.GetClient()
	require.NoError(t, err, "failed getting client")

	address, _, _, err := contract.DeploySimpleStorage(auth, client, big.NewInt(42), "foobar")
	require.NoError(t, err, "failed deploying contract")

	return hexutil.Encode(address[:]), client
}

func createSimulatorAndDeploySimpleStorageContract(ctx context.Context, t *testing.T) (string, bind.ContractBackend) {
	logger := log.GetLogger()
	simulator := NewEthereumSimulatorConnection(logger)
	address, err := simulator.DeployStorageContract(ctx, 42, "foobar")
	require.NoError(t, err, "failed deploying contract to Ethereum")
	client, err := simulator.GetClient()
	require.NoError(t, err, "could not establish Ethereum connection")
	return address, client
}

func callSimpleStorageContractAndAssertReturnedValue(ctx context.Context, address string, client bind.ContractBackend) func(t *testing.T) {
	return func(t *testing.T) {
		parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		require.NoError(t, err, "failed parsing ABI")

		packedInput, err := parsedABI.Pack("getString")
		require.NoError(t, err, "failed packing arguments")

		opts := new(bind.CallOpts)
		msg := ethereum.CallMsg{From: opts.From, To: decodeAddress(address), Data: packedInput}
		packedOutput, err := client.CallContract(ctx, msg, nil)

		var out string
		err = parsedABI.Unpack(&out, "getString", packedOutput)
		require.NoError(t, err, "could not unpack call output")

		require.Equal(t, "foobar", out, "string output differed from expected")
	}
}

func decodeAddress(hexAddress string) *common.Address {
	address, err := hexutil.Decode(hexAddress)
	if err != nil {
		panic(err)
	}
	decoded := common.BytesToAddress(address)

	return &decoded
}

