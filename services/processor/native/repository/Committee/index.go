// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Committee"
const METHOD_GET_ORDERED_COMMITTEE = "getOrderedCommittee"
const METHOD_UPDATE_REPUTATION = "updateReputation"

var PUBLIC = sdk.Export(getOrderedCommittee, updateReputation)
var SYSTEM = sdk.Export(_init)

const ToleranceLevel = uint32(4)
const ReputationBottomCap = uint32(10)

func _init() {
}

func _formatReputation(addr []byte) []byte {
	return []byte(fmt.Sprintf("Validator_%s_Rep", hex.EncodeToString(addr)))
}

func _getReputation(addr []byte) uint32 {
	return state.ReadUint32(_formatReputation(addr))
}

func _degradeReputation(addr []byte) {
	currRep := _getReputation(addr)
	if currRep < ReputationBottomCap {
		state.WriteUint32(_formatReputation(addr), currRep+1)
	}
}

func _clearReputation(addr []byte) {
	state.Clear(_formatReputation(addr))
}

func _getElectedValidators() []byte {
	outputArray := service.CallMethod(elections_systemcontract.CONTRACT_NAME, elections_systemcontract.METHOD_GET_ELECTED_VALIDATORS)
	return outputArray[0].([]byte)
}

func _concat(addresses [][]byte) []byte {
	oneArrayOfAddresses := make([]byte, 0, len(addresses)*digest.NODE_ADDRESS_SIZE_BYTES)
	for _, addr := range addresses {
		oneArrayOfAddresses = append(oneArrayOfAddresses, addr[:]...)
	}
	return oneArrayOfAddresses
}

func _split(oneArrayOfAddresses []byte) [][]byte {
	numAddresses := len(oneArrayOfAddresses) / digest.NODE_ADDRESS_SIZE_BYTES
	res := make([][]byte, numAddresses)
	for i := 0; i < numAddresses; i++ {
		res[i] = oneArrayOfAddresses[digest.NODE_ADDRESS_SIZE_BYTES*i : digest.NODE_ADDRESS_SIZE_BYTES*(i+1)]
	}
	return res
}
