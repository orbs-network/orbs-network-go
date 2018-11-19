package info_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/address"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Info"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(getSignerAddress)

func getSignerAddress() []byte {
	return address.GetSignerAddress()
}
