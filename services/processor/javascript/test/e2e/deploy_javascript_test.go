// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
// +build !race
// +build javascript

package e2e

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/e2e"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDeploymentOfJavascriptContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	if !jsEnabled() {
		t.Skip("JS disabled")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := e2e.NewAppHarness()
		lt := time.Now()
		e2e.PrintTestTime(t, "started", &lt)

		h.WaitUntilTransactionPoolIsReady(t)
		e2e.PrintTestTime(t, "first block committed", &lt)

		counterStart := uint64(time.Now().UnixNano())
		contractName := fmt.Sprintf("JsTest%d", counterStart)

		e2e.PrintTestTime(t, "send deploy - start", &lt)

		DeployJSContractAndRequireSuccess(h, t, e2e.OwnerOfAllSupply, contractName,
			[]byte(`
import { State, Address } from "orbs-contract-sdk/v1";
const key = new Uint8Array([1, 2, 3]);

export function _init() {
	State.writeString(key, "Station to Station")
}

export function get() {
	return 100
}

export function getSignerAddress() {
	return Address.getSignerAddress()
}

export function saveName(value) {
	State.writeString(key, value)
}

export function getName() {
	return State.readString(key)
}
`))

		e2e.PrintTestTime(t, "send deploy - end", &lt)

		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			e2e.PrintTestTime(t, "run query - start", &lt)
			response, err2 := h.RunQuery(e2e.OwnerOfAllSupply.PublicKey(), contractName, "get")
			e2e.PrintTestTime(t, "run query - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0].(uint32) == 100
			}
			return false
		})
		require.True(t, ok, "get counter should return counter start")

		signerAddress, _ := digest.CalcClientAddressOfEd25519PublicKey(e2e.OwnerOfAllSupply.PublicKey())
		ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			e2e.PrintTestTime(t, "run query - start", &lt)
			response, err2 := h.RunQuery(e2e.OwnerOfAllSupply.PublicKey(), contractName, "getSignerAddress")
			e2e.PrintTestTime(t, "run query - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return bytes.Equal(response.OutputArguments[0].([]byte), signerAddress)
			}
			return false
		})
		require.True(t, ok, "getSignerAddress should return signer address")

		ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			response, err := h.RunQuery(e2e.OwnerOfAllSupply.PublicKey(), contractName, "getName")

			if err == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0].(string) == "Station to Station"
			}
			return false
		})
		require.True(t, ok, "getName should return initial state")

		e2e.PrintTestTime(t, "send transaction - start", &lt)
		response, _, err := h.SendTransaction(e2e.OwnerOfAllSupply.PublicKey(), e2e.OwnerOfAllSupply.PrivateKey(), contractName, "saveName", "Diamond Dogs")
		e2e.PrintTestTime(t, "send transaction - end", &lt)

		require.NoError(t, err, "add transaction should not return error")
		require.Equal(t, codec.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus)
		require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult)

		ok = test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			response, err := h.RunQuery(e2e.OwnerOfAllSupply.PublicKey(), contractName, "getName")

			if err == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0].(string) == "Diamond Dogs"
			}
			return false
		})

		require.True(t, ok, "getName should return name")
		e2e.PrintTestTime(t, "done", &lt)

	})
}

func TestDeploymentOfJavascriptContractInteroperableWithGo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	if !jsEnabled() {
		t.Skip("JS disabled")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := e2e.NewAppHarness()
		lt := time.Now()
		e2e.PrintTestTime(t, "started", &lt)

		h.WaitUntilTransactionPoolIsReady(t)
		e2e.PrintTestTime(t, "first block committed", &lt)

		counterStart := uint64(time.Now().UnixNano())
		goContractName := fmt.Sprintf("GoTest%d", counterStart)
		jsContractName := fmt.Sprintf("JsTest%d", counterStart)

		e2e.PrintTestTime(t, "send deploy - start", &lt)

		h.DeployContractAndRequireSuccess(t, e2e.OwnerOfAllSupply, goContractName,
			[]byte(`
package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
)

var PUBLIC = sdk.Export(getValue, throwPanic)
var SYSTEM = sdk.Export(_init)

func _init() {

}

func getValue() uint64 {
	return uint64(100)
}

func throwPanic() uint64 {
	panic("bang!")
}
`))

		DeployJSContractAndRequireSuccess(h, t, e2e.OwnerOfAllSupply, jsContractName,
			[]byte(`
import { Service } from "orbs-contract-sdk/v1";

export function _init() {

}

export function getValue(contractName) {
	return Service.callMethod(contractName, "getValue")
}

export function checkPanic(contractName) {
	return Service.callMethod(contractName, "throwPanic")
}

export function checkNonExistentMethod(contractName) {
	return Service.callMethod(contractName, "methodDoesNotExist")
}
`))

		e2e.PrintTestTime(t, "send deploy - end", &lt)

		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			e2e.PrintTestTime(t, "run query - start", &lt)
			response, err2 := h.RunQuery(e2e.OwnerOfAllSupply.PublicKey(), jsContractName, "getValue", goContractName)
			e2e.PrintTestTime(t, "run query - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return response.OutputArguments[0].(uint64) == 100
			}
			return false
		})
		require.True(t, ok, "getValue() should call the go contract and get a result")

		okWithPanic := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			e2e.PrintTestTime(t, "run query - start", &lt)
			response, _ := h.RunQuery(e2e.OwnerOfAllSupply.PublicKey(), jsContractName, "checkPanic", goContractName)
			e2e.PrintTestTime(t, "run query - end", &lt)

			if response.ExecutionResult == codec.EXECUTION_RESULT_ERROR_SMART_CONTRACT {
				return response.OutputArguments[0].(string) == "bang!"
			}
			return false
		})
		require.True(t, okWithPanic, "throwPanic() should call the go contract and get an error")

		okWithNonExistentMethod := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			e2e.PrintTestTime(t, "run query - start", &lt)
			response, _ := h.RunQuery(e2e.OwnerOfAllSupply.PublicKey(), jsContractName, "checkNonExistentMethod", goContractName)
			e2e.PrintTestTime(t, "run query - end", &lt)

			if response.ExecutionResult == codec.EXECUTION_RESULT_ERROR_SMART_CONTRACT {
				t.Log(response.OutputArguments[0].(string))
				return response.OutputArguments[0].(string) == fmt.Sprintf("method 'methodDoesNotExist' not found on contract '%s'", goContractName)
			}
			return false
		})
		require.True(t, okWithNonExistentMethod, "checkNonExistentMethod() should call the go contract and get an error")
	})
}
