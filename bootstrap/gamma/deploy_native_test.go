//+build !race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func testDeployNativeContractWithConfig(jsonConfig string) func(t *testing.T) {
	return func(t *testing.T) {
		test.WithConcurrencyHarness(t, func(ctx context.Context, harness *test.ConcurrencyHarness) {
			network := NewDevelopmentNetwork(ctx, harness.Logger, nil, jsonConfig)
			harness.Supervise(network)
			contract := callcontract.NewContractClient(network)

			counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

			t.Log("deploying contract")

			contract.DeployNativeCounterContract(ctx, 1, 0) // for benchmark: leader is nodeIndex 0, validator is nodeIndex 1

			require.True(t, test.Eventually(3*time.Second, func() bool {
				return counterStart == contract.CounterGet(ctx, 0)

			}), "expected counter value to equal it's initial value")

			t.Log("transacting with contract")

			contract.CounterAdd(ctx, 1, 17)

			require.True(t, test.Eventually(3*time.Second, func() bool {
				return counterStart+17 == contract.CounterGet(ctx, 0)
			}), "expected counter value to be incremented by transaction")

		})
	}
}

func TestNonLeaderDeploysNativeContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compilation of contracts in short mode")
	}

	t.Run("Benchmark", testDeployNativeContractWithConfig(""))
	t.Run("LeanHelix", testDeployNativeContractWithConfig(fmt.Sprintf(`{"active-consensus-algo":%d}`, consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX)))
}

func TestDeployNativeContractFailsWhenUsingSystemContractName(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, harness *test.ConcurrencyHarness) {
		network := NewDevelopmentNetwork(ctx, harness.Logger, nil, "")
		harness.Supervise(network)
		tx := builders.Transaction().
			WithVirtualChainId(network.VirtualChainId).
			WithMethod("_Deployments", "deployService").
			WithArgs(elections_systemcontract.CONTRACT_NAME,
				uint32(protocol.PROCESSOR_TYPE_NATIVE),
				[]byte(contracts.NativeSourceCodeForCounter(0)),
			).
			WithEd25519Signer(keys.Ed25519KeyPairForTests(0)).
			Builder()

		txResponse, _ := network.SendTransaction(ctx, tx, 0)
		require.EqualValues(t, protocol.REQUEST_STATUS_COMPLETED, txResponse.RequestResult().RequestStatus())
		require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, txResponse.TransactionStatus())
		require.EqualValues(t, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, txResponse.TransactionReceipt().ExecutionResult())
		textValue := builders.PackedArgumentArrayDecode(txResponse.TransactionReceipt().RawOutputArgumentArrayWithHeader()).ArgumentsIterator().NextArguments().StringValue()
		require.EqualValues(t, "a contract with this name exists", textValue)
	})
}
