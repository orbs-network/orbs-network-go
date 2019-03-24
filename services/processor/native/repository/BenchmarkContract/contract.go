// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package benchmarkcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/events"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "BenchmarkContract"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(add, set, get, argTypes, throw, giveBirth)
var SYSTEM = sdk.Export(_init)
var EVENTS = sdk.Export(BabyBorn)

var PRIVATE = sdk.Export(nop) // needed to avoid lint error since this private function is not used by anyone (it's for a test)

func BabyBorn(name string, weight uint32) {}

func _init() {
	state.WriteUint64([]byte("initialized"), 1)
}

func nop() {
}

func add(a uint64, b uint64) uint64 {
	return a + b
}

func set(a uint64) {
	state.WriteUint64([]byte("example-key"), a)
}

func get() uint64 {
	return state.ReadUint64([]byte("example-key"))
}

func argTypes(a1 uint32, a2 uint64, a3 string, a4 []byte) (uint32, uint64, string, []byte) {
	return a1 + 1, a2 + 1, a3 + "1", append(a4, 0x01)
}

func throw() {
	panic("example error returned by contract")
}

func giveBirth(name string) {
	events.EmitEvent(BabyBorn, name, uint32(3))
}
