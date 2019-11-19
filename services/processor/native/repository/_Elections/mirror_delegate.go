// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"math/big"
)

/***
 * Mirror : transfer, delegate
 */
type Transfer struct {
	From  [20]byte
	To    [20]byte
	Value *big.Int
}

func mirrorDelegationByTransfer(hexEncodedEthTxHash string) {
	_initCurrentElection()
	if hasProcessingStarted() == 1 {
		panic(fmt.Errorf("proccessing has started cannot mirror now, resubmit next election"))
	}

	e := &Transfer{}
	eventBlockNumber, eventBlockTxIndex := ethereum.GetTransactionLog(getTokenEthereumContractAddress(), getTokenAbi(), hexEncodedEthTxHash, DELEGATION_BY_TRANSFER_NAME, e)

	if DELEGATION_BY_TRANSFER_VALUE.Cmp(e.Value) != 0 {
		panic(fmt.Errorf("mirrorDelegateByTransfer from %x to %x failed since %d is wrong delegation value", e.From, e.To, e.Value.Uint64()))
	}

	_mirrorDelegateImpl(e.From[:], e.To[:], eventBlockNumber, eventBlockTxIndex, DELEGATION_BY_TRANSFER_NAME)
}

type Delegate struct {
	Delegator [20]byte
	To        [20]byte
}

func mirrorDelegation(hexEncodedEthTxHash string) {
	_initCurrentElection()
	if hasProcessingStarted() == 1 {
		panic(fmt.Errorf("proccessing has started cannot mirror now, resubmit next election"))
	}

	e := &Delegate{}
	eventBlockNumber, eventBlockTxIndex := ethereum.GetTransactionLog(getVotingEthereumContractAddress(), getVotingAbi(), hexEncodedEthTxHash, DELEGATION_NAME, e)

	_mirrorDelegateImpl(e.Delegator[:], e.To[:], eventBlockNumber, eventBlockTxIndex, DELEGATION_NAME)
}

func _mirrorDelegateImpl(delegator []byte, agent []byte, eventBlockNumber uint64, eventBlockTxIndex uint32, eventName string) {
	if _isMirrorDelegationDataAfterElection(eventBlockNumber) {
		panic(fmt.Errorf("delegate with medthod %s from %x to %x failed since it happened in block number %d which is after election date, resubmit next election",
			eventName, delegator, agent, eventBlockNumber))
	}
	_mirrorDelegationData(delegator, agent, eventBlockNumber, eventBlockTxIndex, eventName)
}

func _isMirrorDelegationDataAfterElection(eventBlockNumber uint64) bool {
	if _isTimeBasedElections() {
		if ethereum.GetBlockTimeByNumber(eventBlockNumber) > getCurrentElectionTimeInNanos() {
			return true
		}
	} else {
		temp := getCurrentElectionBlockNumber()
		if eventBlockNumber > temp {
			return true
		}
	}
	return false
}

func _mirrorDelegationData(delegator []byte, agent []byte, eventBlockNumber uint64, eventBlockTxIndex uint32, eventName string) {
	stateMethod := state.ReadString(_formatDelegatorMethod(delegator))
	stateBlockNumber := uint64(0)
	if stateMethod == DELEGATION_NAME && eventName == DELEGATION_BY_TRANSFER_NAME {
		panic(fmt.Errorf("delegate with medthod %s from %x to %x failed since already have delegation with method %s",
			eventName, delegator, agent, stateMethod))
	} else if stateMethod == DELEGATION_BY_TRANSFER_NAME && eventName == DELEGATION_NAME {
		stateBlockNumber = eventBlockNumber
	} else if stateMethod == eventName {
		stateBlockNumber = state.ReadUint64(_formatDelegatorBlockNumberKey(delegator))
		stateBlockTxIndex := state.ReadUint32(_formatDelegatorBlockTxIndexKey(delegator))
		if stateBlockNumber > eventBlockNumber || (stateBlockNumber == eventBlockNumber && stateBlockTxIndex >= eventBlockTxIndex) {
			panic(fmt.Errorf("delegate from %x to %x with block-height %d and tx-index %d failed since current delegation is from block-height %d and tx-index %d",
				delegator, agent, eventBlockNumber, eventBlockTxIndex, stateBlockNumber, stateBlockTxIndex))
		}
	}

	if stateBlockNumber == 0 { // new delegator
		numOfDelegators := _getNumberOfDelegators()
		_setDelegatorAtIndex(numOfDelegators, delegator)
		_setNumberOfDelegators(numOfDelegators + 1)
	}
	emptyAddr := [20]byte{}
	if bytes.Equal(delegator, agent) {
		agent = emptyAddr[:]
	}

	state.WriteBytes(_formatDelegatorAgentKey(delegator), agent)
	state.WriteUint64(_formatDelegatorBlockNumberKey(delegator), eventBlockNumber)
	state.WriteUint32(_formatDelegatorBlockTxIndexKey(delegator), eventBlockTxIndex)
	state.WriteString(_formatDelegatorMethod(delegator), eventName)
}

/***
 * Delegators - Data struct
 */
func _formatNumberOfDelegators() []byte {
	return []byte("Delegator_Address_Count")
}

func _getNumberOfDelegators() int {
	return int(state.ReadUint32(_formatNumberOfDelegators()))
}

func _setNumberOfDelegators(numberOfDelegators int) {
	state.WriteUint32(_formatNumberOfDelegators(), uint32(numberOfDelegators))
}

func _getDelegatorAtIndex(index int) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatDelegatorIterator(index)))
}

func _setDelegatorAtIndex(index int, delegator []byte) {
	state.WriteBytes(_formatDelegatorIterator(index), delegator)
}

func _formatDelegatorIterator(num int) []byte {
	return []byte(fmt.Sprintf("Delegator_Address_%d", num))
}

func _getDelegatorGuardian(delegator []byte) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatDelegatorAgentKey(delegator)))
}

func _formatDelegatorAgentKey(delegator []byte) []byte {
	return []byte(fmt.Sprintf("Delegator_%s_Agent", hex.EncodeToString(delegator)))
}

func _formatDelegatorBlockNumberKey(delegator []byte) []byte {
	return []byte(fmt.Sprintf("Delegator_%s_BlockNumber", hex.EncodeToString(delegator)))
}

func _formatDelegatorBlockTxIndexKey(delegator []byte) []byte {
	return []byte(fmt.Sprintf("Delegator_%s_BlockTxIndex", hex.EncodeToString(delegator)))
}

func _formatDelegatorMethod(delegator []byte) []byte {
	return []byte(fmt.Sprintf("Delegator_%s_Method", hex.EncodeToString(delegator)))
}

func _formatDelegatorStakeKey(delegator []byte) []byte {
	return []byte(fmt.Sprintf("Delegator_%s_Stake", hex.EncodeToString(delegator)))
}
