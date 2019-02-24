package benchmarktoken

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "BenchmarkToken"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(transfer, getBalance)
var SYSTEM = sdk.Export(_init)

const TOTAL_SUPPLY = uint64(10000000000)

func _init() {
	ownerAddress := address.GetSignerAddress()
	state.WriteUint64(ownerAddress, TOTAL_SUPPLY)
}

func transfer(amount uint64, targetAddress []byte) {
	// sender
	callerAddress := address.GetCallerAddress()
	callerBalance := state.ReadUint64(callerAddress)
	if callerBalance < amount {
		panic(fmt.Sprintf("transfer of %d failed since balance is only %d", amount, callerBalance))
	}
	state.WriteUint64(callerAddress, callerBalance-amount)

	// recipient
	address.ValidateAddress(targetAddress)
	targetBalance := state.ReadUint64(targetAddress)
	state.WriteUint64(targetAddress, targetBalance+amount)
}

func getBalance(targetAddress []byte) uint64 {
	address.ValidateAddress(targetAddress)
	return state.ReadUint64(targetAddress)
}
