// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// Contract that shows that contract with public function that accept and return types of bool, big.Int, [20]byte and [32]byte
package fixed_bytes

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"math/big"
)

var PUBLIC = sdk.Export(getAddress, setAddress, getHash, setHash, getBool, setBool, getToken, setToken)
var SYSTEM = sdk.Export(_init)

func _init() {
}

func getAddress() [20]byte {
	return state.ReadBytes20([]byte("bytes20"))
}

func setAddress(addr [20]byte) {
	state.WriteBytes20([]byte("bytes20"), addr)
}

func getHash() [32]byte {
	return state.ReadBytes32([]byte("bytes32"))
}

func setHash(addr [32]byte) {
	state.WriteBytes32([]byte("bytes32"), addr)
}

func getBool() bool {
	return state.ReadBool([]byte("bool"))
}

func setBool(enabled bool) {
	state.WriteBool([]byte("bool"), enabled)
}

func getToken() *big.Int {
	return state.ReadBigInt([]byte("token"))
}

func setToken(token *big.Int) {
	state.WriteBigInt([]byte("token"), token)
}
