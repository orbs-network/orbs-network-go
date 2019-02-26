package test

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
)

func TestEthereumConnector_GetTransactionLogs(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		simulator := adapter.NewEthereumSimulatorConnection(logger)
		auth := simulator.GetAuth()
		connector := ethereum.NewEthereumCrosschainConnector(simulator, logger)

		contractAddress, deployedContract, err := simulator.DeployEthereumContract(auth, contract.EmitEventAbi, contract.EmitEventBin)
		simulator.Commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		amount := big.NewInt(42)
		tuid := big.NewInt(33)
		ethAddress := common.BigToAddress(big.NewInt(42000000000))
		orbsAddress := anOrbsAddress()

		tx, err := deployedContract.Transact(auth, "transferOut", tuid, ethAddress, orbsAddress, amount)
		simulator.Commit()
		require.NoError(t, err, "failed emitting event")

		out, err := connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
			EthereumContractAddress: hexutil.Encode(contractAddress.Bytes()),
			EthereumTxhash:          primitives.Uint256(tx.Hash().Bytes()),
			EthereumEventName:       "TransferredOut",
			EthereumJsonAbi:         contract.EmitEventAbi,
			ReferenceTimestamp:      primitives.TimestampNano(0), //TODO real timestamp
		})
		require.NoError(t, err, "failed getting logs")

		parsedABI, err := abi.JSON(strings.NewReader(contract.EmitEventAbi))
		require.NoError(t, err, "failed parsing ABI")

		event := new(contract.EmitEvent)
		err = ethereum.ABIUnpackAllEventArguments(parsedABI, event, "TransferredOut", out.EthereumAbiPackedOutput)
		require.NoError(t, err, "failed getting amount from tx log")
		require.EqualValues(t, ethAddress, event.EthAddress, "failed getting ethAddress from unpacked data")
		require.EqualValues(t, tuid, event.Tuid, "failed getting tuid from unpacked data")
		require.EqualValues(t, orbsAddress, event.OrbsAddress, "failed getting orbsAddress from unpacked data")
		require.EqualValues(t, amount, event.Value, "failed getting amount from unpacked data")
	})
}

func TestEthereumConnector_GetTransactionLogs_ParsesEventsWithAddressArray(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		simulator := adapter.NewEthereumSimulatorConnection(logger)
		auth := simulator.GetAuth()
		connector := ethereum.NewEthereumCrosschainConnector(simulator, logger)

		contractABI, err := readFile("../contract/EmitAddressArrayEvent_sol_EmitAddressArrayEvent.abi")
		require.NoError(t, err, "failed reading contract ABI")

		contractBin, err := readFile("../contract/EmitAddressArrayEvent_sol_EmitAddressArrayEvent.bin")
		require.NoError(t, err, "failed reading contract binary")

		contractAddress, deployedContract, err := simulator.DeployEthereumContract(auth, string(contractABI), string(contractBin))
		simulator.Commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		addresses := [][20]byte{{0x1, 0x2, 0x3}, {0x4, 0x5, 0x6}}
		tx, err := deployedContract.Transact(auth, "fire", addresses)
		simulator.Commit()
		require.NoError(t, err, "failed emitting event")

		out, err := connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
			EthereumContractAddress: hexutil.Encode(contractAddress.Bytes()),
			EthereumTxhash:          primitives.Uint256(tx.Hash().Bytes()),
			EthereumEventName:       "EventWithAddressArray",
			EthereumJsonAbi:         string(contractABI),
			ReferenceTimestamp:      primitives.TimestampNano(0), //TODO real timestamp
		})
		require.NoError(t, err, "failed getting logs")

		parsedABI, err := abi.JSON(bytes.NewReader(contractABI))
		require.NoError(t, err, "failed parsing ABI")

		event := new(struct {
			Value [][20]byte
		})
		err = ethereum.ABIUnpackAllEventArguments(parsedABI, event, "EventWithAddressArray", out.EthereumAbiPackedOutput)
		require.NoError(t, err, "failed unpacking event")
		require.EqualValues(t, addresses, event.Value, "event did not include expected addresses")
	})
}

func readFile(path string) ([]byte, error) {
	absPath, _ := filepath.Abs(path)
	return ioutil.ReadFile(absPath)
}

func anOrbsAddress() [20]byte {
	orbsUser, _ := orbsClient.CreateAccount()
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], orbsUser.AddressAsBytes())
	return orbsUserAddress
}
