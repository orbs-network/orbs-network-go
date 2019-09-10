package main

import (
	"crypto/sha256"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"golang.org/x/crypto/sha3"
)

var PUBLIC = sdk.Export(sha2_256, sha3_256)
var SYSTEM = sdk.Export(_init)

func _init() {

}

func sha2_256(payload []byte) []byte {
	value := sha256.Sum256(payload)
	return value[:]
}

func sha3_256(payload []byte) []byte {
	value := sha3.Sum256(payload)
	return value[:]
}
