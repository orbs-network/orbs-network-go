package fixed_bytes

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"math/big"
)

const CONTRACT_NAME = "fixedBytes"

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

