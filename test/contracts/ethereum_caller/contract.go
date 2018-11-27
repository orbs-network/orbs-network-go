package ethereum_caller

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
)

var PUBLIC = sdk.Export(readString)
var SYSTEM = sdk.Export(_init)

func _init() {
}

func readString(address string) (out string, err error) {
	ethereum.CallMethod(address, contract.SimpleStorageABI, "getString", out)
	return
}
