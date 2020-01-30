// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Committee"
const METHOD_GET_ORDERED_COMMITTEE = "getOrderedCommittee" // used with election
const METHOD_UPDATE_MISSES = "updateMisses"

var PUBLIC = sdk.Export(getOrderedCommittee, getNextOrderedCommittee, getReputation, getAllCommitteeReputations, getMisses, getAllCommitteeMisses, updateMisses)
var SYSTEM = sdk.Export(_init)
var EVENTS = sdk.Export(CommitteeMemberMissed, CommitteeMemberClosedBlock)

func _init() {
}
