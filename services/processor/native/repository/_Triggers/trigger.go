package triggers_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Triggers"
const METHOD_TRIGGER = "trigger"

var PUBLIC = sdk.Export(trigger)
var SYSTEM = sdk.Export(_init)

func _init() {
}

func trigger() {
	service.CallMethod(elections_systemcontract.CONTRACT_NAME, elections_systemcontract.METHOD_PROCESS_TRIGGER)
}
