package asb_ether

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/events"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/safemath/safeuint64"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
	"math/big"
)

// helpers
const CONTRACT_NAME = "asb_ether"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(setAsbAddr, setTokenContract, resetContract /* TODO V1 security issue*/, getAsbAddr, getAsbAbi, getTokenContract, transferIn, transferOut)
var SYSTEM = sdk.Export(_init, setAsbAbi)
var EVENTS = sdk.Export(OrbsTransferredOut)

// defaults
const TOKEN_CONTRACT_KEY = "_TOKEN_CONTRACT_KEY_"
const defaultTokenContract = "erc20proxy"
const ASB_ETH_ADDR_KEY = "_ASB_ETH_ADDR_KEY_"
const ASB_ABI_KEY = "_ASB_ABI_KEY_"
const defaultAsbAbi = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"bytes20"},{"indexed":false,"name":"value","type":"uint256"}],"name":"EthTransferredOut","type":"event"}]`
const OUT_TUID_KEY = "_OUT_TUID_KEY_"
const IN_TUID_KEY = "_IN_TUID_KEY_"
const IN_TUID_MAX_KEY = "_IN_TUID_MAX_KEY_"

func _init() {
	setAsbAbi(defaultAsbAbi)
	setTokenContract(defaultTokenContract)
	// TODO v1 do we have someway to start with a real asbEthAddress ?
}

type EthTransferredOut struct {
	Tuid  *big.Int
	From  [20]byte
	To    [20]byte
	Value *big.Int
}

func OrbsTransferredOut(
	tuid uint64,
	orbsAddress []byte,
	ethAddress []byte,
	amount uint64) {
}

func transferIn(hexEncodedEthTxHash string) {
	asbAddr := getAsbAddr()
	e := &EthTransferredOut{}
	ethereum.GetTransactionLog(asbAddr, getAsbAbi(), hexEncodedEthTxHash, "EthTransferredOut", e)

	if e.Tuid == nil {
		panic("Got nil tuid from logs")
	}

	if e.Value == nil || e.Value.Cmp(big.NewInt(0)) <= 0 {
		panic("Got nil or non positive value from log")
	}

	address.ValidateAddress(e.To[:])

	inTuidKey := genInTuidKey(e.Tuid.Uint64())
	if isInTuidExists(inTuidKey) {
		panic(fmt.Errorf("transfer of %d to address %x failed since inbound-tuid %d has already been spent", e.Value, e.To, e.Tuid))
	}

	service.CallMethod(getTokenContract(), "mint", e.To[:], e.Value.Uint64())

	setInTuid(inTuidKey)
	setInTuidMax(e.Tuid.Uint64())
}

func transferOut(ethAddr []byte, amount uint64) {
	tuid := safeuint64.Add(getOutTuid(), 1)
	setOutTuid(tuid)

	sourceOrbsAddress := address.GetSignerAddress()
	service.CallMethod(getTokenContract(), "burn", sourceOrbsAddress, amount)

	events.EmitEvent(OrbsTransferredOut, tuid, sourceOrbsAddress, ethAddr, amount)
}

func genInTuidKey(tuid uint64) string {
	return fmt.Sprintf("%s%d", IN_TUID_KEY, tuid)
}

func isInTuidExists(tuid string) bool {
	return state.ReadUint32ByKey(tuid) != 0
}

func setInTuid(tuid string) {
	state.WriteUint32ByKey(tuid, 1)
}

func getInTuidMax() uint64 {
	return state.ReadUint64ByKey(IN_TUID_MAX_KEY)
}

func setInTuidMax(next uint64) {
	state.WriteUint64ByKey(IN_TUID_MAX_KEY, next)
}

func getOutTuid() uint64 {
	return state.ReadUint64ByKey(OUT_TUID_KEY)
}

func setOutTuid(next uint64) {
	state.WriteUint64ByKey(OUT_TUID_KEY, next)
}

func getAsbAddr() string {
	return state.ReadStringByKey(ASB_ETH_ADDR_KEY)
}

func setAsbAddr(asbAddr string) { // upgrade
	state.WriteStringByKey(ASB_ETH_ADDR_KEY, asbAddr)
}

func getAsbAbi() string {
	return state.ReadStringByKey(ASB_ABI_KEY)
}

func setAsbAbi(asbAbi string) { // upgrade
	state.WriteStringByKey(ASB_ABI_KEY, asbAbi)
}

func getTokenContract() string {
	return state.ReadStringByKey(TOKEN_CONTRACT_KEY)
}

func setTokenContract(erc20Proxy string) { // upgrade
	state.WriteStringByKey(TOKEN_CONTRACT_KEY, erc20Proxy)
}

func resetContract() {
	setOutTuid(0)
	max := getInTuidMax()
	for i := uint64(0); i <= max; i++ {
		state.ClearByKey(genInTuidKey(i))
	}
	setInTuidMax(0)
}
