// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"sync"
	"testing"
	"time"

	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestE2EStress(t *testing.T) {
	h := NewAppHarness()
	ctrlRand := rand.NewControlledRand(t)

	generalConfig := GetConfig()
	config := generalConfig.StressTest

	if !config.enabled {
		t.Skip("Skipping stress test")
	}

	baseTxCount := getTransactionCount(t, h)

	var wg sync.WaitGroup

	limiter := rate.NewLimiter(rate.Limit(config.targetTPS), 50)

	var mutex sync.Mutex
	var errors []error
	var errorTransactionStatuses []string

	var clients []*orbsClient.OrbsClient
	for _, apiEndpoint := range config.apiEndpoints {
		clients = append(clients, orbsClient.NewClient(apiEndpoint, uint32(generalConfig.AppVcid), codec.NETWORK_TYPE_TEST_NET))
	}

	defaultTarget, _ := orbsClient.CreateAccount()

	for i := int64(0); i < config.numberOfTransactions; i++ {
		if err := limiter.Wait(context.Background()); err == nil {
			wg.Add(1)

			go func(i int64) {
				defer wg.Done()

				target := defaultTarget
				if !config.skipState {
					target, _ = orbsClient.CreateAccount()
				}

				amount := uint64(ctrlRand.Intn(10))

				client := clients[i%int64(len(clients))] // select one of the clients

				var response *codec.TransactionResponse
				var err2 error

				if config.async {
					response, _, err2 = h.SendTransactionAsyncWithClient(client, OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), "BenchmarkToken", "transfer", amount, target.AddressAsBytes())
				} else {
					response, _, err2 = h.SendTransactionWithClient(client, OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), "BenchmarkToken", "transfer", amount, target.AddressAsBytes())
				}

				if err2 != nil {
					fmt.Println("Encountered an error sending a transaction while stress testing", client.Endpoint, err2)
					mutex.Lock()
					defer mutex.Unlock()
					fmt.Println("")
					errors = append(errors, err2)
					if response != nil {
						errorTransactionStatuses = append(errorTransactionStatuses, string(response.TransactionStatus), "endpoint", client.Endpoint)
					}
				}

				if i+1%100 == 0 {
					fmt.Println(fmt.Sprintf("processed transactions: %d/%d", i+1, config.numberOfTransactions))
				}
			}(i)
		} else {
			mutex.Lock()
			defer mutex.Unlock()
			errors = append(errors, err)
		}
	}

	wg.Wait()

	// very bad and unreliable metric, does not take into account multiple endpoints yet
	txCount := float64(getTransactionCount(t, h) - baseTxCount)

	expectedNumberOfTx := float64(100-config.acceptableFailureRate) / 100 * float64(config.numberOfTransactions)

	fmt.Printf("Successfully processed %.0f%% of transactions\n", txCount/float64(config.numberOfTransactions)*100)

	if len(errors) != 0 {
		fmt.Println()
		fmt.Println("===== ERRORS =====")
		for k, v := range groupErrors(errors) {
			fmt.Printf("%d times: %s\n", v, k)
		}
		fmt.Println("===== ERRORS =====")
		fmt.Println()
		fmt.Println("===== FAILED TX STATUSES =====")
		for k, v := range groupStrings(errorTransactionStatuses) {
			fmt.Printf("%d times: %s\n", v, k)
		}
		fmt.Println("===== FAILED TX STATUSES =====")
		fmt.Println()
	}

	require.Condition(t, func() (success bool) {
		return txCount >= expectedNumberOfTx
	}, "transaction processed (%.0f) < expected transactions processed (%.0f) out of %d transactions sent", txCount, expectedNumberOfTx, config.numberOfTransactions)

	// Commenting out until we get reliable rates

	//ratePerSecond := m["TransactionPool.RatePerSecond"]["RatePerSecond"].(float64)

	//require.Condition(t, func() (success bool) {
	//	return ratePerSecond >= config.targetTPS
	//}, "actual tps (%f) is less than target tps (%f)", ratePerSecond, config.targetTPS)
}

func getTransactionCount(t *testing.T, h *Harness) int64 {
	var txCount int64

	require.True(t, test.Eventually(1*time.Minute, func() bool {
		txCount = h.GetTransactionCount()
		return txCount != 0
	}), "could not retrieve metrics")

	return txCount
}

func groupErrors(errors []error) map[string]int {
	groupedErrors := make(map[string]int)
	for _, error := range errors {
		groupedErrors[error.Error()]++
	}
	return groupedErrors
}

func groupStrings(strings []string) map[string]int {
	groupedStrings := make(map[string]int)
	for _, str := range strings {
		groupedStrings[str]++
	}
	return groupedStrings
}
