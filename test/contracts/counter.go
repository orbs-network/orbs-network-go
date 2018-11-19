package contracts

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/test/contracts/counter_mock"
)

const COUNTER_NATIVE_SOURCE_CODE = `
package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
)

const COUNTER_CONTRACT_START_FROM = uint64(%d)

var PUBLIC = sdk.Export(add, get, start)
var SYSTEM = sdk.Export(_init)

func _init() {
	state.WriteUint64ByKey("count", COUNTER_CONTRACT_START_FROM)
}

func add(amount uint64) {
	count := state.ReadUint64ByKey("count")
	count += amount
	state.WriteUint64ByKey("count", count)
}

func get() uint64 {
	return state.ReadUint64ByKey("count")
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

func MockForCounter() *sdk.ContractInfo {
	return &counter_mock.CONTRACT
}

const MOCK_COUNTER_CONTRACT_START_FROM = counter_mock.COUNTER_CONTRACT_START_FROM
