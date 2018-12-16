package asb_ether

import (
	"encoding/hex"
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

var PUBLIC = sdk.Export(setAsbAddr /* TODO v1 not system*/, getAsbAddr, getAsbAbi, getTokenContract, transferIn, transferOut)
var SYSTEM = sdk.Export(_init, setAsbAbi, setTokenContract)
var EVENTS = sdk.Export(OrbsTransferOut)
var PRIVATE = sdk.Export(getOutTuid, setOutTuid, genInTuidKey, isInTuidExists, setInTuid)

// defaults
const TOKEN_CONTRACT_KEY = "_TOKEN_CONTRACT_KEY_"
const defaultTokenContract = erc20proxy.CONTRACT_NAME
const ASB_ETH_ADDR_KEY = "_ASB_ETH_ADDR_KEY_"
const defaultAsbAddr = "stam" // TODO fill in
const ASB_ABI_KEY = "_ASB_ABI_KEY_"
const defaultAsbAbi = `[{"constant":true,"inputs":[],"name":"orbsASBContractName","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"federation","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"renounceOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"isOwner","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"spentOrbsTuids","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"virtualChainId","outputs":[{"name":"","type":"uint64"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"tuidCounter","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"networkType","outputs":[{"name":"","type":"uint32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"token","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"VERSION","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_networkType","type":"uint32"},{"name":"_virtualChainId","type":"uint64"},{"name":"_orbsASBContractName","type":"string"},{"name":"_token","type":"address"},{"name":"_federation","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"bytes20"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredOut","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"bytes20"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredIn","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"constant":false,"inputs":[{"name":"_to","type":"bytes20"},{"name":"_value","type":"uint256"}],"name":"transferOut","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
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
	amount *big.Int) {
}

func transferIn(hexEncodedEthTxHash string) {
	absAddr := getAsbAddr()
	e := &TransferredOut{}
	ethereum.GetTransactionLog(absAddr, getAsbAbi(), hexEncodedEthTxHash, "TransferredOut", e)

	fmt.Printf("tuid=%s, from=%s, to=%s, value=%s\n", e.Tuid, hex.EncodeToString(e.From.Bytes()), hex.EncodeToString(e.To[:]), e.Value)

	if e.Tuid == nil {
		panic("Got nil tuid from logs")
	}

	if e.Value == nil {
		panic("Got nil value from log")
	}

	inTuidKey := genInTuidKey(e.Tuid.String())
	if isInTuidExists(inTuidKey) {
		panic(fmt.Errorf("transfer of %d to address %x failed since inbound-tuid %d has already been spent", e.Value, e.To, e.Tuid))
	}

	service.CallMethod(getTokenContract(), "mint", e.To[:], e.Value.Uint64()) // todo mint or transfer

	setInTuid(inTuidKey)
}

func transferOut(ethAddr []byte, amount uint64) {
	tuid := safeuint64.Add(getOutTuid(), 1)
	setOutTuid(tuid)

	targetOrbsAddress := address.GetSignerAddress()
	service.CallMethod(getTokenContract(), "burn", targetOrbsAddress, amount) // TODO burn or transfer

	events.EmitEvent(OrbsTransferOut, tuid, ethAddr, targetOrbsAddress, big.NewInt(int64(amount)))
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
