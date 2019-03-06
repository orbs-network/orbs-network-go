package info_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/address"
)

func isAlive() string {
	return "alive"
}

func getSignerAddress() []byte {
	return address.GetSignerAddress()
}
