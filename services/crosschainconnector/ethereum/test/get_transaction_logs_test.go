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
		connector := ethereum.NewEthereumCrosschainConnector(ctx, simulator, logger)

		parsedABI, err := abi.JSON(strings.NewReader(contract.EmitEventAbi))
		require.NoError(t, err, "failed parsing ABI")

		contractAddress, contract, err := simulator.DeployEmitEvent(auth, parsedABI)
		simulator.Commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		amount := big.NewInt(42)
		tx, err := contract.Transact(auth, "transferOut", big.NewInt(0), [20]byte{}, [20]byte{}, amount)
		simulator.Commit()
		require.NoError(t, err, "failed emitting event")

		eventABI := parsedABI.Events["TransferredOut"]

		out, err := connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
			EthereumContractAddress: hexutil.Encode(contractAddress),
			EthereumTxhash:          primitives.Uint256(tx.Hash().Bytes()),
			EventSignature:          string(eventABI.Id().Bytes()),
			ReferenceTimestamp:      primitives.TimestampNano(0), //TODO real timestamp
		})

		require.NoError(t, err, "failed getting logs")

		require.Len(t, out.EthereumPackedEventTopics, 4, "did not get 4 topics from event (expecting 4 topics since event has 3 indexed fields, the first topic being the event signature)")

		outAmount, err := eventABI.Inputs.UnpackValues(out.EthereumPackedEventData)
		require.NoError(t, err, "failed getting amount from tx log")
		require.EqualValues(t, amount, outAmount[0], "failed getting amount from unpacked data")
	})
}
