package usecases

const UseGoroutine = `package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

var PUBLIC = sdk.Export(add)
var SYSTEM = sdk.Export(_init)

var COUNTER_KEY = []byte("count")

func _init() {
	state.WriteUint64(COUNTER_KEY, 0)
}

func add(amount uint64) {
    var i int
	go func() {
		i = 10
	}()

	count := state.ReadUint64(COUNTER_KEY)
	count += amount
	state.WriteUint64(COUNTER_KEY, count)
}
`
