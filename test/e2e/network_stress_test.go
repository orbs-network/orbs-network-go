// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"context"
	"fmt"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"sync"
	"testing"
	"time"
)

func TestE2EStress(t *testing.T) {
	h := newHarness()
	ctrlRand := rand.NewControlledRand(t)

	config := getConfig().stressTest

	if !config.enabled {
		t.Skip("Skipping stress test")
	}

	baseTxCount := getTransactionCount(t, h)

	var wg sync.WaitGroup

	limiter := rate.NewLimiter(1000, 50)

	var mutex sync.Mutex
	var errors []error
	var errorTransactionStatuses []string

	for i := int64(0); i < config.numberOfTransactions; i++ {
		if err := limiter.Wait(context.Background()); err == nil {
			wg.Add(1)

			go func() {
				defer wg.Done()

				target, _ := orbsClient.CreateAccount()
				amount := uint64(ctrlRand.Intn(10))

				response, _, err2 := h.sendTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), "BenchmarkToken", "transfer", uint64(amount), target.AddressAsBytes())

				if err2 != nil {
					mutex.Lock()
					defer mutex.Unlock()
					errors = append(errors, err2)
					errorTransactionStatuses = append(errorTransactionStatuses, string(response.TransactionStatus))
				}
			}()
		} else {
			mutex.Lock()
			defer mutex.Unlock()
			errors = append(errors, err)
		}
	}

	wg.Wait()

	txCount := getTransactionCount(t, h) - baseTxCount

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

	//ratePerSecond := m["TransactionPool.RatePerSecond"]["Rate"].(float64)

	//require.Condition(t, func() (success bool) {
	//	return ratePerSecond >= config.targetTPS
	//}, "actual tps (%f) is less than target tps (%f)", ratePerSecond, config.targetTPS)
}

func getTransactionCount(t *testing.T, h *harness) float64 {
	var m metrics

	require.True(t, test.Eventually(1*time.Minute, func() bool {
		m = h.getMetrics()
		return m != nil
	}), "could not retrieve metrics")

	return m["TransactionPool.CommittedPool.Transactions.Count"]["Value"].(float64)
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
