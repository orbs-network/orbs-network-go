package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

const COUNTER_CONTRACT_START_FROM = uint64(100)

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
