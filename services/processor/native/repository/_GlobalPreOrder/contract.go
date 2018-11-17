package globalpreorder_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

var EXPORTS = sdk.Export(approve)

func _init() {
}

func approve() {
	// TODO: add subscription check here (panic on error)
}
