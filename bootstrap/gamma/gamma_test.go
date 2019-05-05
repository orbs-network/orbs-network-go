// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

const LEAN_HELIX_CONSENSUS_JSON = `{"active-consensus-algo":2}`
const WAIT_FOR_BLOCK_TIMEOUT = 10 * time.Second
const GRACEFUL_SHUTDOWN_TIMEOUT = 3 * time.Second

type metrics map[string]map[string]interface{}

func waitForBlock(endpoint string, targetBlockHeight primitives.BlockHeight) func() bool {
	return func() bool {
		res, err := http.Get(endpoint + "/metrics")
		if err != nil {
			fmt.Println(err)
			return false
		}

		readBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return false
		}
		m := make(metrics)
		json.Unmarshal(readBytes, &m)

		blockHeight := m["BlockStorage.BlockHeight"]["Value"].(float64)
		return primitives.BlockHeight(blockHeight) >= targetBlockHeight
	}
}

func testGammaWithJSONConfig(configJSON string) func(t *testing.T) {
	return func(t *testing.T) {
		randomPort := test.RandomPort()
		serverAddress := fmt.Sprintf("0.0.0.0:%d", randomPort)
		endpoint := fmt.Sprintf("http://%s", serverAddress)
		gammaServer := StartGammaServer(GammaServerConfig{
			serverAddress, false, configJSON, true,
		})
		defer gammaServer.GracefulShutdown(GRACEFUL_SHUTDOWN_TIMEOUT)

		require.True(t, test.Eventually(WAIT_FOR_BLOCK_TIMEOUT, waitForBlock(endpoint, 1)))

		sender, _ := orbsClient.CreateAccount()
		transferTo, _ := orbsClient.CreateAccount()
		client := orbsClient.NewClient(endpoint, 42, codec.NETWORK_TYPE_TEST_NET)

		payload, _, err := client.CreateTransaction(sender.PublicKey, sender.PrivateKey, "BenchmarkToken", "transfer", uint64(671), transferTo.AddressAsBytes())
		require.NoError(t, err)
		response, err := client.SendTransaction(payload)
		require.NoError(t, err)

		require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult)
		require.True(t, test.Eventually(WAIT_FOR_BLOCK_TIMEOUT, waitForBlock(endpoint, 2)))
	}
}

func testGammaWithEmptyBlocks(configJSON string) func(t *testing.T) {
	return func(t *testing.T) {
		randomPort := test.RandomPort()
		serverAddress := fmt.Sprintf("0.0.0.0:%d", randomPort)
		endpoint := fmt.Sprintf("http://%s", serverAddress)
		gammaServer := StartGammaServer(GammaServerConfig{
			serverAddress, false, configJSON, true,
		})
		defer gammaServer.GracefulShutdown(GRACEFUL_SHUTDOWN_TIMEOUT)

		require.True(t, test.Eventually(WAIT_FOR_BLOCK_TIMEOUT, waitForBlock(endpoint, 5)))
	}
}

func TestGamma(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	t.Run("Benchmark", testGammaWithJSONConfig(""))
	t.Run("LeanHelix", testGammaWithJSONConfig(LEAN_HELIX_CONSENSUS_JSON))
}

func TestGammaWithEmptyBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	t.Run("Benchmark", testGammaWithEmptyBlocks(`{"transaction-pool-time-between-empty-blocks":"200ms"}`))
	t.Run("LeanHelix", testGammaWithEmptyBlocks(`{"active-consensus-algo":2,"transaction-pool-time-between-empty-blocks":"200ms"}`))
}
