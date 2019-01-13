package ethereum_caller_mock

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
)

var PUBLIC = sdk.Export(readString)
var SYSTEM = sdk.Export(_init)

func _init() {
}

func readString(address string) string {
	var out string
	ethereum.CallMethod(address, contract.SimpleStorageABI, "getString", &out)
	return out
}
