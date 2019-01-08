package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/orbs-network/orbs-client-sdk-go/orbsclient"
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
)

func TestEthereumConnector_GetTransactionLogs(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger()
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

func anOrbsAddress() [20]byte {
	orbsUser, _ := orbsclient.CreateAccount()
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], orbsUser.AddressAsBytes())
	return orbsUserAddress
}
