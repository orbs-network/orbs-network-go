package ERC20Proxy

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "erc20Pproxy_contract"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(totalSupply, balanceOf, transfer, approve, allowance, transferFrom)
var SYSTEM = sdk.Export(_init)
var PRIVATE = sdk.Export(_transferImpl)

// defaults
const TOTAL_SUPPLY = 1000000000
const OWNER_KEY = "_OWNER_KEY_"
const TOTAL_SUPPLY_KEY = "_TOTAL_SUPPLY_KEY_"

func _init() {
	ownerAddress := address.GetSignerAddress()
	state.WriteUint64ByKey(TOTAL_SUPPLY_KEY, TOTAL_SUPPLY)
	state.WriteBytesByKey(OWNER_KEY, ownerAddress)
	state.WriteUint64ByAddress(ownerAddress, TOTAL_SUPPLY)
}

func totalSupply() uint64 {
	return state.ReadUint64ByKey(TOTAL_SUPPLY_KEY)
}

func transfer(targetAddress []byte, amount uint64) {
	// validations
	signerAddress := address.GetSignerAddress()
	address.ValidateAddress(targetAddress)

	// transfer
	_transferImpl(signerAddress, targetAddress, amount)
}

func balanceOf(targetAddress []byte) uint64 {
	address.ValidateAddress(targetAddress)
	return state.ReadUint64ByAddress(targetAddress)
}

func approve(targetAddress []byte, amount uint64) {
	// create key
	signerAddress := address.GetSignerAddress()
	address.ValidateAddress(targetAddress)
	allowKey := append(signerAddress, targetAddress...)

	state.WriteUint64ByKey(string(allowKey), amount)
}

func allowance(senderAddress []byte, targetAddress []byte) uint64 {
	// create key
	allowKey := append(senderAddress, targetAddress...)
	return state.ReadUint64ByKey(string(allowKey))
}

func transferFrom(senderAddress []byte, targetAddress []byte, amount uint64) {
	// checks
	address.ValidateAddress(senderAddress)
	address.ValidateAddress(targetAddress)
	allowanceBalance := allowance(senderAddress, targetAddress)
	if allowanceBalance < amount {
		panic(fmt.Sprintf("transferFrom of %d from %x to %x failed since allowance balance is only %d", amount, senderAddress, targetAddress, allowanceBalance))
	}

	// transfer
	_transferImpl(senderAddress, targetAddress, amount)
}

func _transferImpl(senderAddress []byte, targetAddress []byte, amount uint64) {
	// sender
	callerBalance := state.ReadUint64ByAddress(senderAddress)
	if callerBalance < amount {
		panic(fmt.Sprintf("transfer of %d from %x to %x failed since balance is only %d", amount, senderAddress, targetAddress, callerBalance))
	}
	state.WriteUint64ByAddress(senderAddress, callerBalance-amount)

	// recipient
	targetBalance := state.ReadUint64ByAddress(targetAddress)
	state.WriteUint64ByAddress(targetAddress, targetBalance+amount)
}
