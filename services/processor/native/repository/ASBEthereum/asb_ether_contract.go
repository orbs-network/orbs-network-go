package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
	"math/big"
)

// helpers
const CONTRACT_NAME = "ABSEther"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(getAsbAddr, getErc20Proxy, transferIn, transferOut)
var SYSTEM = sdk.Export(_init, setAsbAddr, setErc20Proxy)
var PRIVATE = sdk.Export(getOutTuid, setOutTuid)

// defaults
const ABS_ETH_ADDR_KEY = "_ABS_ETH_ADDR_KEY_"
const defaultAbsAddr = "stam" // TODO fill in
const ERC_PROXY_KEY = "_ERC_PROXY_KEY_"
const defaultErc20Proxy = "Erc20ProxyV1" // TODO fill in
const OUT_TUID_KEY = "_OUT_TUID_KEY_"
const IN_TUID_KEY = "_IN_TUID_KEY_"
const defaultAsbABI = `[{"constant":true,"inputs":[],"name":"verifier","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"orbsASBContractName","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"federation","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"renounceOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"isOwner","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"spentOrbsTuids","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"virtualChainId","outputs":[{"name":"","type":"uint64"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"tuidCounter","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"networkType","outputs":[{"name":"","type":"uint32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"token","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"VERSION","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_networkType","type":"uint32"},{"name":"_virtualChainId","type":"uint64"},{"name":"_orbsASBContractName","type":"string"},{"name":"_token","type":"address"},{"name":"_federation","type":"address"},{"name":"_verifier","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"bytes20"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredOut","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"bytes20"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredIn","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"constant":false,"inputs":[{"name":"_to","type":"bytes20"},{"name":"_value","type":"uint256"}],"name":"transferOut","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_resultsBlockHeader","type":"bytes"},{"name":"_resultsBlockProof","type":"bytes"},{"name":"_transactionReceipt","type":"bytes"},{"name":"_transactionReceiptProof","type":"bytes32[]"}],"name":"transferIn","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_verifier","type":"address"}],"name":"setAutonomousSwapProofVerifier","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

func _init() {
	setAsbAddr(defaultAbsAddr)
	setErc20Proxy(defaultErc20Proxy)
}

//event TransferredOut(uint256 indexed tuid, address indexed from, bytes20 indexed to, uint256 value);
type TransferredOut struct {
	tuid  *big.Int
	from  *common.Address
	to    []byte
	value *big.Int
}

func transferIn(hexEndcodedEthTxHash string) {
	absAddr := getAsbAddr()
	ethEvent := &TransferredOut{}
	ethereum.GetTransactionLog(absAddr, defaultAsbABI, hexEndcodedEthTxHash, "TransferredOut", ethEvent)

	inTuidKey := IN_TUID_KEY + ethEvent.tuid.String()
	exists := state.ReadUint32ByKey(inTuidKey)
	if exists != 0 {
		panic(fmt.Errorf("transfer of %d to address %x failed since inbound-tuid %d has already been spent", ethEvent.value, ethEvent.to, ethEvent.tuid))
	}

	erc20lib := getErc20Proxy()
	service.CallMethod(erc20lib, "mint", ethEvent.to, ethEvent.value) // todo mint or transfer

	state.WriteUint32ByKey(inTuidKey, 1)
}

func transferOut(ethAddr []byte, amount uint) {
	tuid := getOutTuid() + 1
	setOutTuid(tuid) // TODO concurrency

	targetOrbsAddress := address.GetSignerAddress()
	// TODO proof := service.GetProof(tuid, ethAddr, absAddr, amount)
	// TODO absAddr := getAsbAddr()
	// TODO CrossChain.SetTransactionLogs(tuid, ethAddr, absAddr, amount, proof)

	erc20lib := getErc20Proxy()
	service.CallMethod(erc20lib, "burn", targetOrbsAddress, amount) // burn or transfer
}

func getOutTuid() uint64 {
	return state.ReadUint64ByKey(OUT_TUID_KEY)
}

func setOutTuid(next uint64) {
	state.WriteUint64ByKey(OUT_TUID_KEY, next)
}

func getAsbAddr() string {
	return state.ReadStringByKey(ABS_ETH_ADDR_KEY)
}

func setAsbAddr(absAddr string) { // upgrade
	state.WriteStringByKey(ABS_ETH_ADDR_KEY, absAddr)
}

func getErc20Proxy() string {
	return state.ReadStringByKey(ERC_PROXY_KEY)
}

func setErc20Proxy(erc20Proxy string) { // upgrade
	state.WriteStringByKey(ERC_PROXY_KEY, erc20Proxy)
}
