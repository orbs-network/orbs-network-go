package counter_mock

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
)

const COUNTER_CONTRACT_START_FROM = uint64(100)

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
