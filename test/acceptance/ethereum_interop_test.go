// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/ethereum/go-ethereum/common/hexutil"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts/ethereum_caller_mock"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// LH: Can only use after enabling Jonathan's (also Noam) feature for finding a block on Eth based on timestamp (find Taiga)
func TestDeployAndCallContractThatCallsEthereum(t *testing.T) {
	newHarness().
		WithLogFilters(log.ExcludeField(internodesync.LogTag), log.ExcludeEntryPoint("tx-pool-sync"), log.ExcludeEntryPoint("TransactionForwarder")).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {

			addressOfContractInEthereum := deployEthereumContract(t, network.EthereumSimulator(), "foobar")
			deployOrbsContractCallingEthereum(ctx, network)

			require.NoError(t, ctx.Err(), "failed deploying the EthereumReader contract")

			readResponse := readStringFromEthereumReaderAt(ctx, network, addressOfContractInEthereum)

			require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, readResponse.QueryResult().ExecutionResult())
			require.EqualValues(t, "foobar", extractStringValueFrom(readResponse))

		})
}

func deployEthereumContract(t testing.TB, simulator *adapter.EthereumSimulator, stringValue string) string {
	addressOfContractInEthereum, err := simulator.DeploySimpleStorageContract(simulator.GetAuth(), stringValue)
	simulator.Commit()
	require.NoError(t, err, "deploy of storage contract failed")
	return hexutil.Encode(addressOfContractInEthereum[:])
}

func extractStringValueFrom(readResponse *client.RunQueryResponse) string {
	argsArray := builders.PackedArgumentArrayDecode(readResponse.QueryResult().RawOutputArgumentArrayWithHeader())
	return argsArray.ArgumentsIterator().NextArguments().StringValue()
}

func readStringFromEthereumReaderAt(ctx context.Context, network *NetworkHarness, address string) *client.RunQueryResponse {
	readQuery := builders.Query().
		WithMethod("EthereumReader", "readString").
		WithArgs(address).
		Builder()
	readResponse := network.RunQuery(ctx, readQuery, 0)
	return readResponse
}

func deployOrbsContractCallingEthereum(parent context.Context, network *NetworkHarness) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	ethereumReaderCode := "foo" // TODO (v1) this junk argument is very confusing
	network.MockContract(&sdkContext.ContractInfo{
		PublicMethods: ethereum_caller_mock.PUBLIC,
		SystemMethods: ethereum_caller_mock.SYSTEM,
		Permission:    sdkContext.PERMISSION_SCOPE_SERVICE,
	}, ethereumReaderCode)
	deployTx := builders.Transaction().
		WithMethod("_Deployments", "deployService").
		WithArgs(
			"EthereumReader",
			uint32(protocol.PROCESSOR_TYPE_NATIVE),
			[]byte(ethereumReaderCode),
		).Builder()

	deployTxHash := digest.CalcTxHash(deployTx.Build().Transaction()) // because the builder isn't thread safe pre-calc txHash

	network.SendTransactionInBackground(ctx, deployTx, 0)
	network.WaitForTransactionInState(ctx, deployTxHash)
}
