// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package main

// TODO(v1): by talkol: this file should not be here, it should be moved to ROOT/test/contracts

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
)

var PUBLIC = sdk.Export(isAlive)
var SYSTEM = sdk.Export(_init)

func _init() {
}

const ABI = `[{"inputs":[{"name":"_intValue","type":"uint256"},{"name":"_stringValue","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"constant":true,"inputs":[],"name":"getInt","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getString","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getValues","outputs":[{"name":"intValue","type":"uint256"},{"name":"stringValue","type":"string"}],"payable":false,"stateMutability":"view","type":"function"}]`
const ADDRESS = "0xC6CF4977465D1889507bed99f1bA20C050192ed7"

func isAlive() string {
	var out string
	ethereum.CallMethod(ADDRESS, ABI, "getString", &out)
	return out
}
