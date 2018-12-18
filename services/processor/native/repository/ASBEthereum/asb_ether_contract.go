package asb_ether

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/events"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/safemath/safeuint64"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/ERC20Proxy"
	"math/big"
)

// helpers
const CONTRACT_NAME = "asb_ether"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(setAsbAddr /* TODO v1 should be system*/, getAsbAddr, getAsbAbi, getTokenContract, transferIn, transferOut)
var SYSTEM = sdk.Export(_init, setAsbAbi, setTokenContract)
var EVENTS = sdk.Export(OrbsTransferOut)

// defaults
const TOKEN_CONTRACT_KEY = "_TOKEN_CONTRACT_KEY_"
const defaultTokenContract = erc20proxy.CONTRACT_NAME
const ASB_ETH_ADDR_KEY = "_ASB_ETH_ADDR_KEY_"
const defaultAsbAddr = "stam" // TODO v1 do we put a default asb_eth_contract here or force setting after init
const ASB_ABI_KEY = "_ASB_ABI_KEY_"
const defaultAsbAbi = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"bytes20"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredOut","type":"event"}]`
const OUT_TUID_KEY = "_OUT_TUID_KEY_"
const IN_TUID_KEY = "_IN_TUID_KEY_"

func _init() {
	setOutTuid(0)
	setAsbAddr(defaultAsbAddr)
	setAsbAbi(defaultAsbAbi)
	setTokenContract(defaultTokenContract)
}

//event TransferredOut(uint256 indexed tuid, address indexed from, bytes20 indexed to, uint256 value);
type TransferredOut struct {
	Tuid  *big.Int
	From  common.Address
	To    [20]byte
	Value *big.Int
}

func OrbsTransferOut(
	tuid uint64,
	ethAddress []byte,
	orbsAddress []byte,
	amount uint64) {
}

func transferIn(hexEncodedEthTxHash string) {
	absAddr := getAsbAddr()
	e := &TransferredOut{}
	ethereum.GetTransactionLog(absAddr, getAsbAbi(), hexEncodedEthTxHash, "TransferredOut", e)

	if e.Tuid == nil {
		panic("Got nil tuid from logs")
	}

	if e.Value == nil || e.Value.Cmp(big.NewInt(0)) <= 0 {
		panic("Got nil or non positive value from log")
	}

	address.ValidateAddress(e.To[:])

	inTuidKey := genInTuidKey(e.Tuid.String())
	if isInTuidExists(inTuidKey) {
		panic(fmt.Errorf("transfer of %d to address %x failed since inbound-tuid %d has already been spent", e.Value, e.To, e.Tuid))
	}

	service.CallMethod(getTokenContract(), "mint", e.To[:], e.Value.Uint64())

	setInTuid(inTuidKey)
}

func transferOut(ethAddr []byte, amount uint64) {
	tuid := safeuint64.Add(getOutTuid(), 1)
	setOutTuid(tuid)

	sourceOrbsAddress := address.GetSignerAddress()
	service.CallMethod(getTokenContract(), "burn", sourceOrbsAddress, amount)

	events.EmitEvent(OrbsTransferOut, tuid, ethAddr, sourceOrbsAddress, amount)
}

func genInTuidKey(tuid string) string {
	return IN_TUID_KEY + tuid
}

func isInTuidExists(tuid string) bool {
	return state.ReadUint32ByKey(tuid) != 0
}

func setInTuid(tuid string) {
	state.WriteUint32ByKey(tuid, 1)
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

func setAsbAddr(absAddr string) { // upgrade
	state.WriteStringByKey(ASB_ETH_ADDR_KEY, absAddr)
}

func getAsbAbi() string {
	return state.ReadStringByKey(ASB_ABI_KEY)
}

func setAsbAbi(absAbi string) { // upgrade
	state.WriteStringByKey(ASB_ABI_KEY, absAbi)
}

func getTokenContract() string {
	return state.ReadStringByKey(TOKEN_CONTRACT_KEY)
}

func setTokenContract(erc20Proxy string) { // upgrade
	state.WriteStringByKey(TOKEN_CONTRACT_KEY, erc20Proxy)
}
