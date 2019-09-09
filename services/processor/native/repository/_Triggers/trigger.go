package triggers_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/Committee"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Triggers"
const METHOD_TRIGGER = "trigger"
const MANAGEMENT_CHAIN = uint32(1100000) // TODO v2 TODO management chain - this needs to be done differently

var PUBLIC = sdk.Export(trigger)
var SYSTEM = sdk.Export(_init)

func _init() {
}

func trigger() {
	service.CallMethod(committee_systemcontract.CONTRACT_NAME, committee_systemcontract.METHOD_UPDATE_MISSES) // must be before elections
	// TODO v2 TODO management chain and election proxy - there should be call to electionValidators contraact to "get elections" where the if will be
	//	if env.GetVirtualChainId() == MANAGEMENT_CHAIN {
	//		service.CallMethod(elections_systemcontract.CONTRACT_NAME, elections_systemcontract.METHOD_PROCESS_TRIGGER)
	//	}
}
