// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package contracts

import (
	"fmt"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/test/contracts/counter_mock"
)

const COUNTER_NATIVE_SOURCE_CODE = `
package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

const COUNTER_CONTRACT_START_FROM = uint64(%d)

var PUBLIC = sdk.Export(add, get, start)
var SYSTEM = sdk.Export(_init)

var COUNTER_KEY = []byte("count")

func _init() {
	state.WriteUint64(COUNTER_KEY, COUNTER_CONTRACT_START_FROM)
}

func add(amount uint64) {
	count := state.ReadUint64(COUNTER_KEY)
	count += amount
	state.WriteUint64(COUNTER_KEY, count)
}

func get() uint64 {
	return state.ReadUint64(COUNTER_KEY)
}

func start() uint64 {
	return COUNTER_CONTRACT_START_FROM
}
`

func NativeSourceCodeForCounter(startFrom uint64) []byte {
	return []byte(fmt.Sprintf(COUNTER_NATIVE_SOURCE_CODE, startFrom))
}

const COUNTER_JAVASCRIPT_SOURCE_CODE = `
class CounterFrom%d {
	
	static _init() {
		$sdk.state.writeUint64ByKey("count", %d);
	}

	static add(amount) {
		let count = $sdk.state.readUint64ByKey("count");
		count += amount;
		$sdk.state.writeUint64ByKey("count", count);
	}

	static get() {
		return $sdk.state.readUint64ByKey("count");
	}

	static start() {
		return %d;
	}

}
`

func JavaScriptSourceCodeForCounter(startFrom uint64) []byte {
	return []byte(fmt.Sprintf(COUNTER_JAVASCRIPT_SOURCE_CODE, startFrom, startFrom, startFrom))
}

func MockForCounter() *sdkContext.ContractInfo {
	return &sdkContext.ContractInfo{
		PublicMethods: counter_mock.PUBLIC,
		SystemMethods: counter_mock.SYSTEM,
		Permission:    sdkContext.PERMISSION_SCOPE_SERVICE,
	}
}

const MOCK_COUNTER_CONTRACT_START_FROM = counter_mock.COUNTER_CONTRACT_START_FROM
