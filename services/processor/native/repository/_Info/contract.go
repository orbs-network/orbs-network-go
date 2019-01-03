package info_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/address"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Info"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(isAlive, getSignerAddress)

func isAlive() string {
	return "alive"
}

func getSignerAddress() []byte {
	return address.GetSignerAddress()
}
