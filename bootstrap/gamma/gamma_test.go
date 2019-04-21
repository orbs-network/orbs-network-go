// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func testGammaWithJSONConfig(configJSON string) func(t *testing.T) {
	return func(t *testing.T) {
		randomPort := test.RandomPort()
		endpoint := fmt.Sprintf("0.0.0.0:%d", randomPort)
		gammaServer := StartGammaServer(endpoint, false, configJSON, false)
		defer gammaServer.GracefulShutdown(10 * time.Second)

		time.Sleep(5 * time.Second) // waiting for txpool to be ready

		sender, _ := orbsClient.CreateAccount()
		transferTo, _ := orbsClient.CreateAccount()
		client := orbsClient.NewClient(fmt.Sprintf("http://%s", endpoint), 42, codec.NETWORK_TYPE_TEST_NET)

		payload, _, err := client.CreateTransaction(sender.PublicKey, sender.PrivateKey, "BenchmarkToken", "transfer", uint64(671), transferTo.AddressAsBytes())
		require.NoError(t, err)
		response, err := client.SendTransaction(payload)
		require.NoError(t, err)

		require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult)
	}
}

func TestGamma(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	t.Run("Benchmark", testGammaWithJSONConfig("{}"))
	t.Run("LeanHelix", testGammaWithJSONConfig(`{"active-consensus-algo":2}`))
}
