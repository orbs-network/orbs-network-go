// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsVotingContract_isMirrorDelegationDataAfterElection_Before_blockBased(t *testing.T) {
	eventBlockNumber := uint64(200000)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber_InTests(eventBlockNumber + 500)

		// call
		returnValue := _isMirrorDelegationDataAfterElection(eventBlockNumber)
		//assert
		require.False(t, returnValue, "_isMirrorDelegationDataAfterElection returned true for a block that is before the election block")
	})
}

func TestOrbsVotingContract_isMirrorDelegationDataAfterElection_After_blockBased(t *testing.T) {
	eventBlockNumber := uint64(200000)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber_InTests(eventBlockNumber - 500)

		// call
		returnValue := _isMirrorDelegationDataAfterElection(eventBlockNumber)
		//assert
		require.True(t, returnValue, "_isMirrorDelegationDataAfterElection returned false for a block that is after the election block")
	})
}

func TestOrbsVotingContract_mirrorDelegateImpl_EventBlockNumberAfterElectionBlockNumber(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentAddr := []byte{0x02}
	eventName := "Txt"
	eventBlockNumber := uint64(200000)
	eventBlockTxIndex := uint32(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber_InTests(eventBlockNumber - 500)

		//assert
		require.Panics(t, func() {
			_mirrorDelegateImpl(delegatorAddr, agentAddr, eventBlockNumber, eventBlockTxIndex, eventName)
		}, "should panic because event is too new")
	})
}

func TestOrbsVotingContract_mirrorDelegation_AllGood_blockBased(t *testing.T) {
	txHex := "0xabcd"
	delegatorAddr := [20]byte{0x01}
	agentAddr := [20]byte{0x02}
	blockNumber := 100000
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber_InTests(uint64(blockNumber + 1))

		// prepare
		m.MockEthereumLog(getVotingEthereumContractAddress(), getVotingAbi(), txHex, DELEGATION_NAME, blockNumber, txIndex, func(out interface{}) {
			v := out.(*Delegate)
			v.Delegator = delegatorAddr
			v.To = agentAddr
		})

		mirrorDelegation(txHex)

		// assert
		m.VerifyMocks()
		require.EqualValues(t, agentAddr[:], state.ReadBytes(_formatDelegatorAgentKey(delegatorAddr[:])))
		require.EqualValues(t, blockNumber, state.ReadUint64(_formatDelegatorBlockNumberKey(delegatorAddr[:])))
		require.EqualValues(t, txIndex, state.ReadUint32(_formatDelegatorBlockTxIndexKey(delegatorAddr[:])))
		require.EqualValues(t, DELEGATION_NAME, state.ReadString(_formatDelegatorMethod(delegatorAddr[:])))
	})
}

func TestOrbsVotingContract_mirrorDelegationByTransfer_AllGood_blockBased(t *testing.T) {
	txHex := "0xabcd"
	delegatorAddr := [20]byte{0x01}
	agentAddr := [20]byte{0x02}
	value := DELEGATION_BY_TRANSFER_VALUE
	blockNumber := 100000
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber_InTests(uint64(blockNumber + 1))

		// prepare
		m.MockEthereumLog(getTokenEthereumContractAddress(), getTokenAbi(), txHex, DELEGATION_BY_TRANSFER_NAME, blockNumber, txIndex, func(out interface{}) {
			v := out.(*Transfer)
			v.From = delegatorAddr
			v.To = agentAddr
			v.Value = value
		})

		// call
		mirrorDelegationByTransfer(txHex)

		// assert
		m.VerifyMocks()
		require.EqualValues(t, agentAddr[:], state.ReadBytes(_formatDelegatorAgentKey(delegatorAddr[:])))
		require.EqualValues(t, blockNumber, state.ReadUint64(_formatDelegatorBlockNumberKey(delegatorAddr[:])))
		require.EqualValues(t, txIndex, state.ReadUint32(_formatDelegatorBlockTxIndexKey(delegatorAddr[:])))
		require.EqualValues(t, DELEGATION_BY_TRANSFER_NAME, state.ReadBytes(_formatDelegatorMethod(delegatorAddr[:])))
	})
}
