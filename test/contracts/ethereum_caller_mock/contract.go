// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
