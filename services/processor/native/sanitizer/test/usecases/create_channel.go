package usecases

const CreateChannel = `package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

var PUBLIC = sdk.Export(add)
var SYSTEM = sdk.Export(_init)

func _init() {
}

func add(amount uint64) {
	c := make(chan bool)
	c <- true
}
`
