package triggers_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/Committee"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Triggers"
const METHOD_TRIGGER = "trigger"

var PUBLIC = sdk.Export(trigger)
var SYSTEM = sdk.Export(_init)

func _init() {
}

func trigger() {
	service.CallMethod(committee_systemcontract.CONTRACT_NAME, committee_systemcontract.METHOD_UPDATE_MISSES) // committee always refers to the current block's validators - so if this block contains election the result will only affect next block update committee
}
