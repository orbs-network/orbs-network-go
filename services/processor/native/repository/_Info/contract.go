package info_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/address"
)

var EXPORTS = sdk.Export(getSignerAddress)

func _init() {
}

func getSignerAddress() []byte {
	return address.GetSignerAddress()
}
