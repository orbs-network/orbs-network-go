package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
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
		logger := log.GetLogger()
		cfg := &ethereumConnectorConfigForTests{endpoint: "localhost:8545"}
		simulator := NewEthereumSimulatorConnection(cfg, logger)

		address, err := simulator.DeployStorageContract(ctx, 42, "foobar")
		require.NoError(t, err, "failed deploying contract to Ethereum")

		parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		require.NoError(t, err, "failed parsing ABI")

		packedInput, err := parsedABI.Pack("getString")
		require.NoError(t, err, "failed packing arguments")

		opts := new(bind.CallOpts)
		client, err := simulator.GetClient()
		require.NoError(t, err, "could not establish Ethereum connection")
		msg := ethereum.CallMsg{From: opts.From, To: decodeAddress(address), Data: packedInput}
		packedOutput, err := client.CallContract(ctx, msg, nil)

		var out string
		err = parsedABI.Unpack(&out, "getString", packedOutput)
		require.NoError(t, err, "could not unpack call output")

		require.Equal(t, "foobar", out, "string output differed from expected")
	})
}

func decodeAddress(hexAddress string) *common.Address {
	address, err := hexutil.Decode(hexAddress)
	if err != nil {
		panic(err)
	}
	decoded := common.BytesToAddress(address)

	return &decoded
}

