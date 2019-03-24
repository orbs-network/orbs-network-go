// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNonLeaderDeploysNativeContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compilation of contracts in short mode")
	}

	test.WithContext(func(ctx context.Context) {
		network := NewDevelopmentNetwork(ctx, log.DefaultTestingLogger(t), "")
		contract := callcontract.NewContractClient(network)

		counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

		t.Log("deploying contract")

		contract.DeployNativeCounterContract(ctx, 1, 0) // leader is nodeIndex 0, validator is nodeIndex 1

		require.True(t, test.Eventually(3*time.Second, func() bool {
			return counterStart == contract.CounterGet(ctx, 0)

		}), "expected counter value to equal it's initial value")

		t.Log("transacting with contract")

		contract.CounterAdd(ctx, 1, 17)

		require.True(t, test.Eventually(3*time.Second, func() bool {
			return counterStart+17 == contract.CounterGet(ctx, 0)
		}), "expected counter value to be incremented by transaction")

	})
	time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
}
