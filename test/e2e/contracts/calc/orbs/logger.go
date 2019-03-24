// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package main

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"strings"
)

var PUBLIC = sdk.Export(sum, bind)
var SYSTEM = sdk.Export(_init)

func _init() {
}

var ethAddressKey = []byte("ETH_CONTRACT_ADDRESS")
var ethABIKey = []byte("ETH_CONTRACT_ABI")

func bind(ethContractAddress []byte, abi []byte) {
	state.WriteString(ethAddressKey, "0x"+hex.EncodeToString(ethContractAddress))
	state.WriteString(ethABIKey, string(abi))
}

func sum(txCommaSeparatedList string) uint64 {
	abi := state.ReadString(ethABIKey)
	address := state.ReadString(ethAddressKey)
	if abi == "" || address == "" {
		panic("Trying to read from an unbound contract")
	}

	var sum uint64
	for _, txHash := range strings.Split(txCommaSeparatedList, ",") {
		var out struct {
			Count int32
		}

		ethereum.GetTransactionLog(address, abi, txHash, "Log", &out)
		sum += uint64(out.Count)
	}

	return sum
}
