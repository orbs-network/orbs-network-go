package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
	"time"
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
			EthereumContractAddress: contractAddress.Hex(),
			EthereumTxhash:          tx.Hash().Hex(),
			EthereumEventName:       "TransferredOut",
			EthereumJsonAbi:         contract.EmitEventAbi,
			ReferenceTimestamp:      primitives.TimestampNano(time.Now().UnixNano()),
		})
		require.NoError(t, err, "failed getting logs")

		parsedABI, err := abi.JSON(strings.NewReader(contract.EmitEventAbi))
		require.NoError(t, err, "failed parsing ABI")

		event := new(contract.EmitEvent)
		err = ethereum.ABIUnpackAllEventArguments(parsedABI, event, "TransferredOut", out.EthereumAbiPackedOutputs[0])
		require.NoError(t, err, "failed getting amount from tx log")
		require.EqualValues(t, ethAddress, event.EthAddress, "failed getting ethAddress from unpacked data")
		require.EqualValues(t, tuid, event.Tuid, "failed getting tuid from unpacked data")
		require.EqualValues(t, orbsAddress, event.OrbsAddress, "failed getting orbsAddress from unpacked data")
		require.EqualValues(t, amount, event.Value, "failed getting amount from unpacked data")
		require.NotZero(t, out.EthereumBlockNumber, "expected returned block number to be non-zero")
		require.Equal(t, uint32(0), out.EthereumTxindex, "expected returned txIndex to zero")
	})
}

func anOrbsAddress() [20]byte {
	orbsUser, _ := orbsClient.CreateAccount()
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], orbsUser.AddressAsBytes())
	return orbsUserAddress
}
