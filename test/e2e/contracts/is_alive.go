package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/ethereum"
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
