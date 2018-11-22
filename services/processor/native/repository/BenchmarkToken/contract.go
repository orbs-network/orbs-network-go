package benchmarktoken

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "BenchmarkToken"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(transfer, getBalance)
var SYSTEM = sdk.Export(_init)

const TOTAL_SUPPLY = 1000000

func _init() {
	ownerAddress := address.GetSignerAddress()
	state.WriteUint64ByAddress(ownerAddress, TOTAL_SUPPLY)
}

func transfer(amount uint64, targetAddress []byte) {
	// sender
	callerAddress := address.GetCallerAddress()
	callerBalance := state.ReadUint64ByAddress(callerAddress)
	if callerBalance < amount {
		panic(fmt.Sprintf("transfer of %d failed since balance is only %d", amount, callerBalance))
	}
	state.WriteUint64ByAddress(callerAddress, callerBalance-amount)

	// recipient
	address.ValidateAddress(targetAddress)
	targetBalance := state.ReadUint64ByAddress(targetAddress)
	state.WriteUint64ByAddress(targetAddress, targetBalance+amount)
}

func getBalance(targetAddress []byte) uint64 {
	address.ValidateAddress(targetAddress)
	return state.ReadUint64ByAddress(targetAddress)
}
