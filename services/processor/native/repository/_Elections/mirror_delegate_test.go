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
	"math/big"
	"testing"
)

func TestOrbsVotingContract_mirrorDelegationData(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentAddr := []byte{0x02}
	eventName := "Txt"
	eventBlockNumber := uint64(100000)
	eventBlockTxIndex := uint32(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_mirrorDelegationData(delegatorAddr, agentAddr, eventBlockNumber, eventBlockTxIndex, eventName)

		// assert
		require.EqualValues(t, agentAddr[:], state.ReadBytes(_formatDelegatorAgentKey(delegatorAddr[:])))
		require.EqualValues(t, eventBlockNumber, state.ReadUint64(_formatDelegatorBlockNumberKey(delegatorAddr)))
		require.EqualValues(t, eventBlockTxIndex, state.ReadUint32(_formatDelegatorBlockTxIndexKey(delegatorAddr)))
		require.EqualValues(t, eventName, state.ReadBytes(_formatDelegatorMethod(delegatorAddr)))
	})
}

func TestOrbsVotingContract_mirrorDelegationDataTwice(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentAddr := []byte{0x02}
	eventName := "Txt"
	eventBlockNumber := uint64(100000)
	eventBlockTxIndex := uint32(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_mirrorDelegationData(delegatorAddr, agentAddr, eventBlockNumber, eventBlockTxIndex, eventName)

		require.Panics(t, func() {
			_mirrorDelegationData(delegatorAddr, agentAddr, eventBlockNumber, eventBlockTxIndex, eventName)
		}, "should panic because same infor twice")
	})
}

func TestOrbsVotingContract_mirrorDelegationData_TransferDoesNotReplaceDelegate(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentAddr := []byte{0x02}
	eventName := DELEGATION_BY_TRANSFER_NAME
	eventBlockNumber := uint64(100000)
	eventBlockTxIndex := uint32(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		state.WriteString(_formatDelegatorMethod(delegatorAddr), DELEGATION_NAME)

		require.Panics(t, func() {
			_mirrorDelegationData(delegatorAddr, agentAddr, eventBlockNumber, eventBlockTxIndex, eventName)
		}, "should panic because newer delegate")
	})
}

func TestOrbsVotingContract_mirrorDelegationData_DelegateReplacesTransfer(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentOrgAddr := []byte{0x03}
	agentAddr := []byte{0x02}
	eventName := DELEGATION_NAME
	eventBlockNumber := uint64(100000)
	eventBlockTxIndex := uint32(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		_setDelegatorAtIndex(0, delegatorAddr)
		_setNumberOfDelegators(1)
		state.WriteBytes(_formatDelegatorAgentKey(delegatorAddr[:]), agentOrgAddr)
		state.WriteString(_formatDelegatorMethod(delegatorAddr), DELEGATION_BY_TRANSFER_NAME)
		state.WriteUint64(_formatDelegatorBlockNumberKey(delegatorAddr), eventBlockNumber+5)
		state.WriteUint32(_formatDelegatorBlockTxIndexKey(delegatorAddr), 50)

		// call
		_mirrorDelegationData(delegatorAddr, agentAddr, eventBlockNumber, eventBlockTxIndex, eventName)

		// assert
		require.Equal(t, 1, _getNumberOfDelegators())
		require.EqualValues(t, agentAddr[:], state.ReadBytes(_formatDelegatorAgentKey(delegatorAddr)))
		require.EqualValues(t, eventBlockNumber, state.ReadUint64(_formatDelegatorBlockNumberKey(delegatorAddr)))
		require.EqualValues(t, eventBlockTxIndex, state.ReadUint32(_formatDelegatorBlockTxIndexKey(delegatorAddr)))
		require.EqualValues(t, DELEGATION_NAME, state.ReadBytes(_formatDelegatorMethod(delegatorAddr)))
	})
}

func TestOrbsVotingContract_mirrorDelegationData_DelegateReset(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentAddr := []byte{0x02}
	eventName := DELEGATION_NAME
	eventBlockNumber := uint64(100000)
	eventBlockTxIndex := uint32(10)
	emptyAddre := [20]byte{}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		state.WriteBytes(_formatDelegatorAgentKey(delegatorAddr), agentAddr)
		state.WriteString(_formatDelegatorMethod(delegatorAddr), DELEGATION_NAME)
		state.WriteUint64(_formatDelegatorBlockNumberKey(delegatorAddr), eventBlockNumber-5)
		state.WriteUint32(_formatDelegatorBlockTxIndexKey(delegatorAddr), 1)

		// call
		_mirrorDelegationData(delegatorAddr, delegatorAddr, eventBlockNumber, eventBlockTxIndex, eventName)

		// assert
		require.EqualValues(t, emptyAddre[:], state.ReadBytes(_formatDelegatorAgentKey(delegatorAddr)))
		require.EqualValues(t, eventBlockNumber, state.ReadUint64(_formatDelegatorBlockNumberKey(delegatorAddr)))
		require.EqualValues(t, eventBlockTxIndex, state.ReadUint32(_formatDelegatorBlockTxIndexKey(delegatorAddr)))
		require.EqualValues(t, DELEGATION_NAME, state.ReadBytes(_formatDelegatorMethod(delegatorAddr)))
	})
}

func TestOrbsVotingContract_mirrorDelegationData_AlreadyHaveNewerEventBlockNumber(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentAddr := []byte{0x02}
	eventName := "Txt"
	eventBlockNumber := uint64(100000)
	eventBlockTxIndex := uint32(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		state.WriteString(_formatDelegatorMethod(delegatorAddr), eventName)
		state.WriteUint64(_formatDelegatorBlockNumberKey(delegatorAddr), eventBlockNumber+1)

		require.Panics(t, func() {
			_mirrorDelegationData(delegatorAddr, agentAddr, eventBlockNumber, eventBlockTxIndex, eventName)
		}, "should panic because newer block")
	})
}

func TestOrbsVotingContract_mirrorDelegationData_AlreadyHaveNewerEventBlockTxIndex(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentAddr := []byte{0x02}
	eventName := "Txt"
	eventBlockNumber := uint64(100000)
	eventBlockTxIndex := uint32(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		state.WriteString(_formatDelegatorMethod(delegatorAddr), eventName)
		state.WriteUint64(_formatDelegatorBlockNumberKey(delegatorAddr), eventBlockNumber)
		state.WriteUint32(_formatDelegatorBlockTxIndexKey(delegatorAddr), 50)

		require.Panics(t, func() {
			_mirrorDelegationData(delegatorAddr, agentAddr, eventBlockNumber, eventBlockTxIndex, eventName)
		}, "should panic because newer tx index")
	})
}

func TestOrbsVotingContract_isMirrorDelegationDataAfterElection_Before(t *testing.T) {
	eventBlockNumber := 200000

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		electionTime := startTimeBasedGetElectionTime()
		m.MockEthereumGetBlockTimeByNumber(eventBlockNumber, int(electionTime)-10)

		// call
		returnValue := _isMirrorDelegationDataAfterElection(uint64(eventBlockNumber))

		//assert
		require.False(t, returnValue, "_isMirrorDelegationDataAfterElection returned true for a block that is before the election time")
	})
}

func TestOrbsVotingContract_isMirrorDelegationDataAfterElection_After(t *testing.T) {
	eventBlockNumber := 200000

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		electionTime := startTimeBasedGetElectionTime()
		m.MockEthereumGetBlockTimeByNumber(eventBlockNumber, int(electionTime)+10)

		// call
		returnValue := _isMirrorDelegationDataAfterElection(uint64(eventBlockNumber))

		//assert
		require.True(t, returnValue, "_isMirrorDelegationDataAfterElection returned false for a block that is before the election time")
	})
}

func TestOrbsVotingContract_mirrorDelegateImpl_EventAfterElectionTime(t *testing.T) {
	delegatorAddr := []byte{0x01}
	agentAddr := []byte{0x02}
	eventName := "Txt"
	eventBlockNumber := 200000
	eventBlockTxIndex := uint32(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		electionTime := startTimeBasedGetElectionTime()
		m.MockEthereumGetBlockTimeByNumber(eventBlockNumber, int(electionTime)+10)

		//assert
		require.Panics(t, func() {
			_mirrorDelegateImpl(delegatorAddr, agentAddr, uint64(eventBlockNumber), eventBlockTxIndex, eventName)
		}, "should panic because event is too new")
	})
}

func TestOrbsVotingContract_mirrorDelegation_processStarted(t *testing.T) {
	txHex := "0xabcd"
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		// prepare
		_setVotingProcessState("x")

		require.Panics(t, func() {
			mirrorDelegation(txHex)
		}, "should panic because mirror period should have ended")
	})
}

func TestOrbsVotingContract_mirrorDelegationByTransfer_processStarted(t *testing.T) {
	txHex := "0xabcd"
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		// prepare
		_setVotingProcessState("x")
		require.Panics(t, func() {
			mirrorDelegationByTransfer(txHex)
		}, "should panic because mirror period should have ended")
	})
}

func TestOrbsVotingContract_mirrorDelegation(t *testing.T) {
	txHex := "0xabcd"
	delegatorAddr := [20]byte{0x01}
	agentAddr := [20]byte{0x02}
	blockNumber := 100000
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		electionTime := startTimeBasedGetElectionTime()
		m.MockEthereumGetBlockTimeByNumber(blockNumber, int(electionTime)-10)
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

func TestOrbsVotingContract_mirrorDelegationByTransfer(t *testing.T) {
	txHex := "0xabcd"
	delegatorAddr := [20]byte{0x01}
	agentAddr := [20]byte{0x02}
	value := DELEGATION_BY_TRANSFER_VALUE
	blockNumber := 100000
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		electionTime := startTimeBasedGetElectionTime()
		m.MockEthereumGetBlockTimeByNumber(blockNumber, int(electionTime)-10)
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

func TestOrbsVotingContract_mirrorDelegationByTransfer_WrongValue(t *testing.T) {
	txHex := "0xabcd"
	value := big.NewInt(8)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		startTimeBasedGetElectionTime()
		m.MockEthereumLog(getTokenEthereumContractAddress(), getTokenAbi(), txHex, DELEGATION_BY_TRANSFER_NAME, 100, 10, func(out interface{}) {
			v := out.(*Transfer)
			v.Value = value
		})

		// call
		require.Panics(t, func() {
			mirrorDelegationByTransfer(txHex)
		}, "should panic because bad transfer value")
	})
}
