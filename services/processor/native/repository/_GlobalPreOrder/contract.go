package globalpreorder_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_GlobalPreOrder"
const METHOD_APPROVE = "approve"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(approve)

func approve() {
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/572): add subscription check here (panic on error)
}
