package globalpreorder_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_GlobalPreOrder"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(approve)

func approve() {
	// TODO: add subscription check here (panic on error)
}
