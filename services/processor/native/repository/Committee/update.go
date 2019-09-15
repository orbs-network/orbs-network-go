// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/env"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/events"
)

const TRIGGER_CONTRACT = "_Triggers" // hard coded to avoid recursive import

func updateMisses() {
	if !bytes.Equal(address.GetCallerAddress(), address.GetContractAddress(TRIGGER_CONTRACT)){
		panic(fmt.Errorf("must be called from %s contract only", TRIGGER_CONTRACT))
	}
	elected := _split(_getElectedValidators())
	ordered := _getOrderedCommitteeArray(elected)
	blockProposer := env.GetBlockProposerAddress()

	if !_isMemberOfOrderedCommittee(ordered, blockProposer) {
		panic(fmt.Errorf("block proposer address from %x was not found in committee of block height %d", blockProposer, env.GetBlockHeight()))
	}
	_updateMissesByCommitteeOrder(ordered, blockProposer)
}

/*
 The separation between isMember and update is because panic of transaction deletes ContractStateDiff but not Events Emitted.
 So first need to see if blockProposer is in the committee (if not panic) and only then do the update
 */
func _isMemberOfOrderedCommittee(orderedCommittee [][]byte, blockProposer []byte) bool {
	for _, member := range orderedCommittee {
		if bytes.Equal(member, blockProposer) {
			return true
		}
	}
	return false
}

func CommitteeMemberClosedBlock(address []byte) {}
func CommitteeMemberMissed(address []byte) {}
func _updateMissesByCommitteeOrder(orderedCommittee [][]byte, blockProposer []byte)  {
	for _, member := range orderedCommittee {
		if bytes.Equal(member, blockProposer) {
			_clearMiss(member)
			events.EmitEvent(CommitteeMemberClosedBlock, member)
			break
		} else {
			_addMiss(member)
			events.EmitEvent(CommitteeMemberMissed, member)
		}
	}
}
