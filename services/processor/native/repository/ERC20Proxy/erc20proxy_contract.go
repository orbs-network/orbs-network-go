package erc20proxy

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "erc20proxy"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(totalSupply, balanceOf, transfer, approve, allowance, transferFrom, asbBind, asbGetAddress, asbMint, asbBurn)
var SYSTEM = sdk.Export(_init)

// defaults
const OWNER_KEY = "_OWNER_KEY_"
const TOTAL_SUPPLY_KEY = "_TOTAL_SUPPLY_KEY_"
const ASB_ADDR_KEY = "_ASB_ADDR_KEY_"

func _init() {
	ownerAddress := address.GetSignerAddress()
	//state.WriteUint64ByKey(TOTAL_SUPPLY_KEY, 0)
	state.WriteBytesByKey(OWNER_KEY, ownerAddress)
}

func totalSupply() uint64 {
	return state.ReadUint64ByKey(TOTAL_SUPPLY_KEY)
}

func transfer(to []byte, amount uint64) {
	// validations
	callerAddress := address.GetCallerAddress()
	address.ValidateAddress(to)

	// transfer
	_transferImpl(callerAddress, to, amount)
}

func balanceOf(addr []byte) uint64 {
	address.ValidateAddress(addr)
	return state.ReadUint64ByAddress(addr)
}

func _allowKey(addr1 []byte, addr2 []byte) string {
	return string(append(addr1, addr2...))
}

func approve(spenderAddress []byte, amount uint64) {
	callerAddress := address.GetCallerAddress()
	address.ValidateAddress(spenderAddress)

	state.WriteUint64ByKey(_allowKey(callerAddress, spenderAddress), amount)
}

func allowance(from []byte, spenderAddress []byte) uint64 {
	return state.ReadUint64ByKey(_allowKey(from, spenderAddress))
}

func transferFrom(from []byte, to []byte, amount uint64) {
	// checks
	spenderAddress := address.GetCallerAddress()
	address.ValidateAddress(from)
	address.ValidateAddress(to)
	allowanceBalance := allowance(from, spenderAddress)
	if allowanceBalance < amount {
		panic(fmt.Sprintf("transferFrom of %d from %x to %x failed since allowance balance of spender %x is only %d", amount, from, to, spenderAddress, allowanceBalance))
	}

	// reduce allowance
	state.WriteUint64ByKey(_allowKey(from, spenderAddress), allowanceBalance-amount)
	// transfer
	_transferImpl(from, to, amount)
}

func _transferImpl(from []byte, to []byte, amount uint64) {
	// sender
	balance := state.ReadUint64ByAddress(from)
	if balance < amount {
		panic(fmt.Sprintf("transfer of %d from %x to %x failed since balance is only %d", amount, from, to, balance))
	}
	state.WriteUint64ByAddress(from, balance-amount)

	// recipient
	targetBalance := state.ReadUint64ByAddress(to)
	state.WriteUint64ByAddress(to, targetBalance+amount)
}

func asbMint(targetAddress []byte, amount uint64) {
	if !bytes.Equal(asbGetAddress(), address.GetCallerAddress()) {
		panic("only asb contract can call asbMint")
	}
	address.ValidateAddress(targetAddress)
	targetBalance := state.ReadUint64ByAddress(targetAddress)
	state.WriteUint64ByAddress(targetAddress, targetBalance+amount)
	total := state.ReadUint64ByKey(TOTAL_SUPPLY_KEY)
	state.WriteUint64ByKey(TOTAL_SUPPLY_KEY, total+amount)
}

func asbBurn(targetAddress []byte, amount uint64) {
	if !bytes.Equal(asbGetAddress(), address.GetCallerAddress()) {
		panic("only asb contract can call asbBurn")
	}
	address.ValidateAddress(targetAddress)
	targetBalance := state.ReadUint64ByAddress(targetAddress)
	if targetBalance < amount {
		panic(fmt.Sprintf("burn of %d from %x failed since balance is only %d", amount, targetAddress, targetBalance))
	}
	state.WriteUint64ByAddress(targetAddress, targetBalance-amount)
	total := state.ReadUint64ByKey(TOTAL_SUPPLY_KEY)
	state.WriteUint64ByKey(TOTAL_SUPPLY_KEY, total-amount)
}

func asbBind(asbAddress []byte) {
	owner := state.ReadBytesByKey(OWNER_KEY)
	caller := address.GetCallerAddress()
	if !bytes.Equal(owner, caller) {
		panic("only owner can call asbBind")
	}
	state.WriteBytesByKey(ASB_ADDR_KEY, asbAddress)
}

func asbGetAddress() []byte {
	return state.ReadBytesByKey(ASB_ADDR_KEY)
}
