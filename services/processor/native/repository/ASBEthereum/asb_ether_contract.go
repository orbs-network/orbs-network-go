package asb_ether

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/events"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/safemath/safeuint64"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/ERC20Proxy"
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
const defaultTokenContract = erc20proxy.CONTRACT_NAME
const defaultAsbAbi = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"bytes20"},{"indexed":false,"name":"value","type":"uint256"}],"name":"EthTransferredOut","type":"event"}]`

// state keys
var TOKEN_CONTRACT_KEY = []byte("_TOKEN_CONTRACT_KEY_")
var ASB_ETH_ADDR_KEY = []byte("_ASB_ETH_ADDR_KEY_")
var ASB_ABI_KEY = []byte("_ASB_ABI_KEY_")
var OUT_TUID_KEY = []byte("_OUT_TUID_KEY_")
var IN_TUID_KEY = []byte("_IN_TUID_KEY_")
var IN_TUID_MAX_KEY = []byte("_IN_TUID_KEY_")

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

	inTuidKey := genInTuidKey(e.Tuid.Bytes())
	if isInTuidExists(inTuidKey) {
		panic(fmt.Errorf("transfer of %d to address %x failed since inbound-tuid %d has already been spent", e.Value, e.To, e.Tuid))
	}

	service.CallMethod(getTokenContract(), "asbMint", e.To[:], e.Value.Uint64())

	setInTuid(inTuidKey)
	setInTuidMax(e.Tuid.Uint64())
}

func transferOut(ethAddr []byte, amount uint64) {
	tuid := safeuint64.Add(getOutTuid(), 1)
	setOutTuid(tuid)

	sourceOrbsAddress := address.GetSignerAddress()
	service.CallMethod(getTokenContract(), "asbBurn", sourceOrbsAddress, amount)

	events.EmitEvent(OrbsTransferredOut, tuid, sourceOrbsAddress, ethAddr, amount)
}

func genInTuidKey(tuid []byte) []byte {
	return append(IN_TUID_KEY, tuid...)
}

func isInTuidExists(tuidKey []byte) bool {
	return state.ReadUint32(tuidKey) != 0
}

func setInTuid(tuidKey []byte) {
	state.WriteUint32(tuidKey, 1)
}

func getInTuidMax() uint64 {
	return state.ReadUint64(IN_TUID_MAX_KEY)
}

func setInTuidMax(next uint64) {
	state.WriteUint64(IN_TUID_MAX_KEY, next)
}

func getOutTuid() uint64 {
	return state.ReadUint64(OUT_TUID_KEY)
}

func setOutTuid(next uint64) {
	state.WriteUint64(OUT_TUID_KEY, next)
}

func getAsbAddr() string {
	return state.ReadString(ASB_ETH_ADDR_KEY)
}

func setAsbAddr(asbAddr string) { // upgrade
	state.WriteString(ASB_ETH_ADDR_KEY, asbAddr)
}

func getAsbAbi() string {
	return state.ReadString(ASB_ABI_KEY)
}

func setAsbAbi(asbAbi string) { // upgrade
	state.WriteString(ASB_ABI_KEY, asbAbi)
}

func getTokenContract() string {
	return state.ReadString(TOKEN_CONTRACT_KEY)
}

func setTokenContract(erc20Proxy string) { // upgrade
	state.WriteString(TOKEN_CONTRACT_KEY, erc20Proxy)
}

func resetContract() {
	setOutTuid(0)
	max := int64(getInTuidMax())
	for i := int64(0); i <= max; i++ {
		state.Clear(genInTuidKey(big.NewInt(i).Bytes()))
	}
	setInTuidMax(0)
}
