// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//
// +build !race
// +build javascript

package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/contracts/counter_mock"
	"github.com/orbs-network/orbs-network-go/test/e2e"
	"testing"
	"time"

	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
)

func TestDeploymentOfProcessorPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := e2e.NewAppHarness()
		lt := time.Now()
		e2e.PrintTestTime(t, "started", &lt)

		h.WaitUntilTransactionPoolIsReady(t)
		e2e.PrintTestTime(t, "first block committed", &lt)

		counterStart := counter_mock.COUNTER_CONTRACT_START_FROM
		contractName := fmt.Sprintf("CounterFrom%d", time.Now().UnixNano())

		e2e.PrintTestTime(t, "send deploy - start", &lt)

		DeployJSContractAndRequireSuccess(h, t, e2e.OwnerOfAllSupply, contractName,
			[]byte("this contract is fake"))

		e2e.PrintTestTime(t, "send deploy - end", &lt)

		// check counter
		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			e2e.PrintTestTime(t, "run query - start", &lt)
			response, err2 := h.RunQuery(e2e.OwnerOfAllSupply.PublicKey(), contractName, "get")
			e2e.PrintTestTime(t, "run query - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0].(uint64) == counterStart
			}
			return false
		})
		require.True(t, ok, "get counter should return counter start")
		e2e.PrintTestTime(t, "done", &lt)
	})
}
