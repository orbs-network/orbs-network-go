// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/stretchr/testify/require"
)

func TestDeploymentOfNativeContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := NewAppHarness()
		lt := time.Now()
		PrintTestTime(t, "started", &lt)

		h.WaitUntilTransactionPoolIsReady(t)
		PrintTestTime(t, "first block committed", &lt)

		counterStart := uint64(time.Now().UnixNano())
		contractName := fmt.Sprintf("CounterFrom%d", counterStart)
		fmt.Println("Will attempt to deploy contract with name: ", contractName)
		PrintTestTime(t, "send deploy - start", &lt)

		h.DeployContractAndRequireSuccess(t, OwnerOfAllSupply, contractName,
			contracts.NativeSourceCodeForCounterPart1(counterStart),
			contracts.NativeSourceCodeForCounterPart2(counterStart))

		PrintTestTime(t, "send deploy - end", &lt)

		// check counter
		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			PrintTestTime(t, "run query - start", &lt)
			response, err2 := h.RunQuery(OwnerOfAllSupply.PublicKey(), contractName, "get")
			PrintTestTime(t, "run query - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0] == counterStart
			}
			return false
		})
		require.True(t, ok, "get counter should return counter start")

		// transaction to add to the counter
		amount := uint64(17)

		PrintTestTime(t, "send transaction - start", &lt)
		response, _, err := h.SendTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), contractName, "add", uint64(amount))
		PrintTestTime(t, "send transaction - end", &lt)

		require.NoError(t, err, "add transaction should not return error")
		requireSuccessful(t, response)

		// check counter
		ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			response, err := h.RunQuery(OwnerOfAllSupply.PublicKey(), contractName, "get")

			if err == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0] == counterStart+amount
			}
			return false
		})

		require.True(t, ok, "get counter should return counter start plus added value")

		PrintTestTime(t, "attempting to deploy again to assert we can't deploy the same contract twice", &lt)

		response, err = h.DeployNativeContract(OwnerOfAllSupply, contractName, []byte("some other code"))
		require.NoError(t, err, "deployment transaction should fail but not return an error")
		require.EqualValues(t, codec.EXECUTION_RESULT_ERROR_SMART_CONTRACT, response.ExecutionResult, "expected deploy contract to fail")

		PrintTestTime(t, "done", &lt)

	})
}
