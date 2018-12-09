package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

		parsedABI, err := abi.JSON(strings.NewReader(contract.EmitEventAbi))
		require.NoError(t, err, "failed parsing ABI")

		contractAddress, deployedContract, err := simulator.DeployEmitEvent(auth, parsedABI)
		simulator.Commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		amount := big.NewInt(42)
		tuid := big.NewInt(33)
		ethAddress := [20]byte{0x01, 0x02, 0x03}
		orbsAddress := [20]byte{0x04, 0x05, 0x06}

		tx, err := deployedContract.Transact(auth, "transferOut", tuid, ethAddress, orbsAddress, amount)
		simulator.Commit()
		require.NoError(t, err, "failed emitting event")

		out, err := connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
			EthereumContractAddress: hexutil.Encode(contractAddress),
			EthereumTxhash:          primitives.Uint256(tx.Hash().Bytes()),
			EthereumEventName:       "TransferredOut",
			EthereumJsonAbi:         contract.EmitEventAbi,
			ReferenceTimestamp:      primitives.TimestampNano(0), //TODO real timestamp
		})
		require.NoError(t, err, "failed getting logs")

		outArgs, err := ethereum.ABIUnpackAllEventArgumentsValues(parsedABI, "TransferredOut", out.EthereumAbiPackedOutput)
		require.NoError(t, err, "failed getting amount from tx log")
		require.EqualValues(t, tuid, outArgs[0], "failed getting tuid from unpacked data")
		require.EqualValues(t, ethAddress, outArgs[1], "failed getting ethAddress from unpacked data")
		require.EqualValues(t, orbsAddress, outArgs[2], "failed getting orbsAddress from unpacked data")
		require.EqualValues(t, amount, outArgs[3], "failed getting amount from unpacked data")
	})
}
