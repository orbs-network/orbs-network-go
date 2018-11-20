package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
)

var PUBLIC = sdk.Export(add, get, start)
var SYSTEM = sdk.Export(_init)

func _init() {
	state.WriteUint64ByKey("count", 0)
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
	return 0
}
