// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Committee"
const METHOD_GET_ORDERED_COMMITTEE = "getOrderedCommittee"
const METHOD_GET_ORDERED_COMMITTEE_FOR_ADDRESSES = "getOrderedCommitteeForAddresses"
const METHOD_UPDATE_MISSES = "updateMisses"

var PUBLIC = sdk.Export(getOrderedCommittee, getOrderedCommitteeForAddresses, getReputation, getMisses, updateMisses)
var SYSTEM = sdk.Export(_init)
var EVENTS = sdk.Export(CommitteeMemberMissed, CommitteeMemberClosedBlock)

func _init() {
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
