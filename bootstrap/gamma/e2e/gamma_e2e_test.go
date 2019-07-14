// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func testGammaWithJSONConfig(configJSON string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := runGammaOnRandomPort(t, configJSON)

		sender, _ := orbsClient.CreateAccount()
		transferTo, _ := orbsClient.CreateAccount()
		client := orbsClient.NewClient(endpoint, 42, codec.NETWORK_TYPE_TEST_NET)

		sendTransaction(t, client, sender, "BenchmarkToken", "transfer", uint64(671), transferTo.AddressAsBytes())

		require.True(t, test.Eventually(WAIT_FOR_BLOCK_TIMEOUT, waitForBlock(endpoint, 2)))
	}
}

func testGammaWithEmptyBlocks(configJSON string) func(t *testing.T) {
	return func(t *testing.T) {
		endpoint := runGammaOnRandomPort(t, configJSON)

		require.True(t, test.Eventually(WAIT_FOR_BLOCK_TIMEOUT, waitForBlock(endpoint, 5)))
	}
}

func TestGamma(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	t.Run("Benchmark", testGammaWithJSONConfig(""))
	t.Run("LeanHelix", testGammaWithJSONConfig(fmt.Sprintf(`{"active-consensus-algo":%d}`, consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX)))
}

func TestGammaWithEmptyBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	t.Run("Benchmark", testGammaWithEmptyBlocks(`{"transaction-pool-time-between-empty-blocks":"200ms"}`))
	t.Run("LeanHelix", testGammaWithEmptyBlocks(fmt.Sprintf(`{"active-consensus-algo":%d,"transaction-pool-time-between-empty-blocks":"200ms"}`, consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX)))
}

func TestGammaSetBlockTime(t *testing.T) {
	endpoint := runGammaOnRandomPort(t, "")

	timeTravel(t, endpoint, 10*time.Second)

	sender, _ := orbsClient.CreateAccount()
	transferTo, _ := orbsClient.CreateAccount()
	client := orbsClient.NewClient(endpoint, 42, codec.NETWORK_TYPE_TEST_NET)

	desiredTime := time.Now().Add(10 * time.Second)
	response := sendTransaction(t, client, sender, "BenchmarkToken", "transfer", uint64(671), transferTo.AddressAsBytes())

	require.WithinDuration(t, desiredTime, response.BlockTimestamp, 1*time.Second, "new block time did not increase") // we check within a delta to prevent flakiness
}
