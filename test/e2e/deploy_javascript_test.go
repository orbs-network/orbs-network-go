//+build !race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDeploymentOfJavascriptContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := newAppHarness()
		lt := time.Now()
		printTestTime(t, "started", &lt)

		h.waitUntilTransactionPoolIsReady(t)
		printTestTime(t, "first block committed", &lt)

		counterStart := uint64(time.Now().UnixNano())
		contractName := fmt.Sprintf("CounterFrom%d", counterStart)

		printTestTime(t, "send deploy - start", &lt)

		h.deployJSContractAndRequireSuccess(t, OwnerOfAllSupply, contractName,
			[]byte(`
function get() {
	return 100
}

function getSignerAddress() {
	return Address.GetSignerAddress()
}
`))

		printTestTime(t, "send deploy - end", &lt)

		//// check counter
		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			printTestTime(t, "run query - start", &lt)
			response, err2 := h.runQuery(OwnerOfAllSupply.PublicKey(), contractName, "get")
			printTestTime(t, "run query - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0].(uint32) == 100 //counterStart
			}
			return false
		})
		require.True(t, ok, "get counter should return counter start")

		signerAddress, _ := digest.CalcClientAddressOfEd25519PublicKey(OwnerOfAllSupply.PublicKey())
		ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			printTestTime(t, "run query - start", &lt)
			response, err2 := h.runQuery(OwnerOfAllSupply.PublicKey(), contractName, "getSignerAddress")
			printTestTime(t, "run query - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				fmt.Println("YYY", hex.EncodeToString(response.OutputArguments[0].([]byte)))
				fmt.Println("NNN", hex.EncodeToString(OwnerOfAllSupply.PublicKey()))
				return bytes.Equal(response.OutputArguments[0].([]byte), signerAddress)
			}
			return false
		})
		require.True(t, ok, "get counter should return counter start")

		//// check counter
		//ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		//	printTestTime(t, "run query - start", &lt)
		//	response, err2 := h.runQuery(OwnerOfAllSupply.PublicKey(), contractName, "get")
		//	printTestTime(t, "run query - end", &lt)
		//
		//	if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
		//		return response.OutputArguments[0] == 100 //counterStart
		//	}
		//	return false
		//})
		//require.True(t, ok, "get counter should return counter start")
		//
		//// transaction to add to the counter
		//amount := uint64(17)
		//
		//printTestTime(t, "send transaction - start", &lt)
		//response, _, err := h.sendTransaction(OwnerOfAllSupply.PublicKey(), OwnerOfAllSupply.PrivateKey(), contractName, "add", uint64(amount))
		//printTestTime(t, "send transaction - end", &lt)
		//
		//require.NoError(t, err, "add transaction should not return error")
		//require.Equal(t, codec.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus)
		//require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult)
		//
		//// check counter
		//ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		//	response, err := h.runQuery(OwnerOfAllSupply.PublicKey(), contractName, "get")
		//
		//	if err == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
		//		return response.OutputArguments[0] == counterStart+amount
		//	}
		//	return false
		//})
		//
		//require.True(t, ok, "get counter should return counter start plus added value")
		//printTestTime(t, "done", &lt)

	})
}
