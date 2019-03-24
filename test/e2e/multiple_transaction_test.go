// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNetworkCommitsMultipleTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := newHarness()
		lt := time.Now()
		printTestTime(t, "started", &lt)

		h.waitUntilTransactionPoolIsReady(t)
		printTestTime(t, "first block committed", &lt)

		transferTo, _ := orbsClient.CreateAccount()

		// send 3 transactions with total of 70
		amounts := []uint64{15, 22, 33}
		txIds := []string{}
		for _, amount := range amounts {
			printTestTime(t, "send transaction - start", &lt)
			response, txId, err := h.sendTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), "BenchmarkToken", "transfer", uint64(amount), transferTo.AddressAsBytes())
			printTestTime(t, "send transaction - end", &lt)

			txIds = append(txIds, txId)
			require.NoError(t, err, "transaction for amount %d should not return error\nresponse: %+v", amount, response)
			require.Equal(t, codec.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus)
			require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult)
		}

		// get statuses and receipt proofs
		for _, txId := range txIds {
			printTestTime(t, "get status - start", &lt)
			response, err := h.getTransactionStatus(txId)
			printTestTime(t, "get status - end", &lt)

			require.NoError(t, err, "get status for txid %s should not return error", txId)
			require.Equal(t, codec.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus)
			require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult)

			printTestTime(t, "get receipt proof - start", &lt)
			proofResponse, err := h.getTransactionReceiptProof(txId)
			printTestTime(t, "get receipt proof - end", &lt)

			require.NoError(t, err, "get receipt proof for txid %s should not return error", txId)
			require.Equal(t, codec.TRANSACTION_STATUS_COMMITTED, proofResponse.TransactionStatus)
			require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, proofResponse.ExecutionResult)
			require.True(t, len(proofResponse.PackedProof) > 20, "packed receipt proof for txid %s should return at least 20 bytes", txId)
		}

		// check balance
		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			printTestTime(t, "run query - start", &lt)
			response, err := h.runQuery(transferTo.PublicKey, "BenchmarkToken", "getBalance", transferTo.AddressAsBytes())
			printTestTime(t, "run query - end", &lt)

			if err == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0] == uint64(70)
			}
			return false
		})

		require.True(t, ok, "getBalance should return total amount")
		printTestTime(t, "done", &lt)

	})
}
