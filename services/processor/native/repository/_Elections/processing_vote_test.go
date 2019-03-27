// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/safemath/safeuint64"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestOrbsVotingContract_processVoting_MirroringPeriodNotEnded(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setTimingInMirror(m)

		require.Panics(t, func() {
			processVoting()
		}, "should panic because mirror not done")
	})
}

func TestOrbsVotingContract_getStakeFromEthereum(t *testing.T) {
	addr := [20]byte{0x01}
	blockNumber := uint64(100)
	stakeSetup := 64

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		_setCurrentElectionBlockNumber(blockNumber)
		mockStakeInEthereum(m, blockNumber, addr, stakeSetup)

		// call
		stake := _getStakeAtElection(addr)

		// assert
		m.VerifyMocks()
		require.EqualValues(t, stakeSetup, stake)
	})
}

func TestOrbsVotingContract_processVote_CalulateStakes(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	h.blockNumber = h.electionBlock + VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS + 2
	aRecentVoteBlock := h.electionBlock - 1
	anAncientVoteBlock := h.electionBlock - 2*VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS - 2

	var v1, v2, v3, v4, v5 = h.addValidator(), h.addValidator(), h.addValidator(), h.addValidator(), h.addValidator()
	var g1, g2, g3, g4, g5 = h.addGuardian(100), h.addGuardian(200), h.addGuardian(400), h.addGuardian(1000), h.addGuardian(10000000)

	g1.vote(aRecentVoteBlock, v2, v1)
	g2.vote(aRecentVoteBlock, v2, v1)
	g3.vote(aRecentVoteBlock, v2, v3)
	g4.vote(aRecentVoteBlock, v2, v5)
	g5.vote(anAncientVoteBlock, v4)

	for i := 0; i < 10; i++ {
		h.addDelegator(500, g3.address)
	}

	d1 := h.addDelegator(500, g4.address)
	d2 := h.addDelegator(500, d1.address)
	h.addDelegator(500, d2.address)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupEthereumStateBeforeProcess(m)
		h.setupOrbsStateBeforeProcess()

		// call
		expectedNumOfStateTransitions := len(h.guardians) + len(h.delegators) + len(h.validators) + 2
		elected, actualRuns := h.runProcessVoteNtimes(expectedNumOfStateTransitions)

		// assert
		m.VerifyMocks()
		require.True(t, actualRuns <= expectedNumOfStateTransitions, "did not finish in correct amount of passes")
		require.EqualValues(t, "", _getVotingProcessState())
		require.ElementsMatch(t, [][20]byte{v1.address, v3.address, v4.address, v5.address}, elected)
		require.EqualValues(t, 40, getCumulativeParticipationReward(d2.address[:]))
		require.EqualValues(t, 8, getCumulativeParticipationReward(g1.address[:]))
		require.EqualValues(t, 80, getCumulativeParticipationReward(g4.address[:]))
		require.EqualValues(t, 16, getCumulativeParticipationReward(g2.address[:]))
		require.EqualValues(t, 32, getCumulativeParticipationReward(g3.address[:]))
	})
}

func TestOrbsVotingContract_processVote_CalulateStakes_GuardianIsNotGuardian(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	h.blockNumber = h.electionBlock + VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS + 2
	aRecentVoteBlock := h.electionBlock - 1

	var v1 = h.addValidator()
	var g1, g2 = h.addGuardian(1000), h.addGuardian(100).withIsGuardian(false)

	g1.vote(aRecentVoteBlock, v1)
	g2.vote(aRecentVoteBlock, v1)

	d1 := h.addDelegator(1000, g1.address)
	h.addDelegator(1000, g1.address)
	d3 := h.addDelegator(100, g2.address)
	h.addDelegator(100, g2.address)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupEthereumStateBeforeProcess(m)
		h.setupOrbsStateBeforeProcess()

		// call
		h.runProcessVoteNtimes(0)

		// assert
		m.VerifyMocks()
		require.EqualValues(t, 3000, getGuardianVotingWeight(g1.address[:]))
		require.True(t, 0 != getCumulativeParticipationReward(d1.address[:]))
		require.EqualValues(t, 0, getGuardianVotingWeight(g2.address[:]))
		require.EqualValues(t, 0, getCumulativeParticipationReward(d3.address[:]))
	})
}

func TestOrbsVotingContract_processVote_CalulateStakes_DelegatorIsGuardian(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	h.blockNumber = h.electionBlock + VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS + 2
	aRecentVoteBlock := h.electionBlock - 1

	v1 := h.addValidator()
	g1, g2, g3, g4 := h.addGuardian(10), h.addGuardian(100), h.addGuardian(1000), h.addGuardian(10000000)
	fakeD1 := h.addGuardian(10000)
	fakeD2 := h.addGuardian(100000)
	fakeD3 := h.addGuardian(1000000)

	g1.vote(aRecentVoteBlock, v1)
	g2.vote(aRecentVoteBlock, v1)
	g3.vote(aRecentVoteBlock, v1)
	g4.vote(aRecentVoteBlock, v1)

	// direct delegate from delegator that is guardian to a guardian
	d1 := &delegator{actor: actor{stake: fakeD1.stake, address: fakeD1.address}, delegate: g1.address}
	h.delegators = append(h.delegators, d1)

	// delegate from delegator that is guardian to real delegator to a guardian
	realD2 := h.addDelegator(1, g2.address)
	d2 := &delegator{actor: actor{stake: fakeD2.stake, address: fakeD2.address}, delegate: realD2.address}
	h.delegators = append(h.delegators, d2)

	// delegate from real delegator to delegator that is guardian to a guardian
	d3 := &delegator{actor: actor{stake: fakeD3.stake, address: fakeD3.address}, delegate: g3.address}
	h.delegators = append(h.delegators, d3)
	realD3 := h.addDelegator(1, fakeD3.address)

	// direct delegate from delegator that is guardian who voted to a guardian
	d4 := &delegator{actor: actor{stake: g4.stake, address: g4.address}, delegate: g1.address}
	h.delegators = append(h.delegators, d4)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupEthereumValidatorsBeforeProcess(m)
		mockGuardiansInEthereum(m, h.electionBlock, h.guardians)
		h.setupEthereumGuardiansDataBeforeProcess(m)
		mockStakeInEthereum(m, h.electionBlock, realD2.address, realD2.stake)
		mockStakeInEthereum(m, h.electionBlock, realD3.address, realD3.stake)
		//		h.setupEthereumStateBeforeProcess(m)
		h.setupOrbsStateBeforeProcess()

		// call
		expectedNumOfStateTransitions := len(h.guardians) + len(h.delegators) + len(h.validators) + 3
		_, actualRuns := h.runProcessVoteNtimes(expectedNumOfStateTransitions)

		// assert
		//m.VerifyMocks()
		require.True(t, actualRuns <= expectedNumOfStateTransitions, "did not finish in correct amount of passes")
		require.EqualValues(t, 10, getGuardianVotingWeight(g1.address[:]))
		require.EqualValues(t, 101, getGuardianVotingWeight(g2.address[:]))
		require.EqualValues(t, 1000, getGuardianVotingWeight(g3.address[:]))
		require.EqualValues(t, 10000000, getGuardianVotingWeight(g4.address[:]))
		require.EqualValues(t, 0, getGuardianVotingWeight(fakeD1.address[:]))
		require.EqualValues(t, 0, getGuardianVotingWeight(fakeD2.address[:]))
		require.EqualValues(t, 0, getGuardianVotingWeight(fakeD3.address[:]))
		require.EqualValues(t, 0, getCumulativeParticipationReward(realD3.address[:]))
		require.EqualValues(t, 1, state.ReadUint64(_formatDelegatorStakeKey(realD2.address[:])))
		require.EqualValues(t, 1, state.ReadUint64(_formatDelegatorStakeKey(realD3.address[:])))
	})
}

func TestOrbsVotingContract_processVote_ValidatorsFromEthereumToState(t *testing.T) {
	electionBlock := uint64(60000)
	validators := [][20]byte{{0x01}, {0x02}}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		mockValidatorsInEthereum(m, electionBlock, validators)
		_setCurrentElectionBlockNumber(electionBlock)

		// call
		_readValidatorsFromEthereumToState()
		stateValidators := _getValidators()

		// assert
		m.VerifyMocks()
		require.EqualValues(t, len(validators), _getNumberOfValidators())
		for i := 0; i < _getNumberOfValidators(); i++ {
			require.EqualValues(t, validators[i], _getValidatorEthereumAddressAtIndex(i))
		}
		require.EqualValues(t, len(validators), len(stateValidators))
		for i := 0; i < len(validators); i++ {
			require.EqualValues(t, validators[i], stateValidators[i])
		}
	})
}

func TestOrbsVotingContract_processVote_collectValidatorDataFromEthereum(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)

	var v1, v2 = h.addValidatorWithStake(100), h.addValidatorWithStake(200)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()
		mockValidatorOrbsAddressInEthereum(m, h.electionBlock, v1.address, v1.orbsAddress)
		mockStakeInEthereum(m, h.electionBlock, v1.address, 250)
		mockValidatorOrbsAddressInEthereum(m, h.electionBlock, v2.address, v2.orbsAddress)
		mockStakeInEthereum(m, h.electionBlock, v2.address, 450)
		_setVotingProcessItem(0)

		// call
		i := 0
		for ; i < 2; i++ {
			_collectNextValidatorDataFromEthereum()
		}

		// assert
		m.VerifyMocks()
		require.EqualValues(t, i, _getVotingProcessItem())
		require.EqualValues(t, 250, state.ReadUint64(_formatValidatorStakeKey(v1.address[:])))
		require.EqualValues(t, v1.orbsAddress[:], state.ReadBytes(_formatValidatorOrbsAddressKey(v1.address[:])))
		require.EqualValues(t, 450, state.ReadUint64(_formatValidatorStakeKey(v2.address[:])))
		require.EqualValues(t, v2.orbsAddress[:], state.ReadBytes(_formatValidatorOrbsAddressKey(v2.address[:])))
	})
}

func TestOrbsVotingContract_processVote_readGuardiansFromEthereum(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)

	var g1, g2 = h.addGuardian(100), h.addGuardian(200)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		mockGuardiansInEthereum(m, h.electionBlock, h.guardians)
		_setCurrentElectionBlockNumber(h.electionBlock)
		_setGuardianStake(_formatGuardianStakeKey(g1.address[:]), 500)

		// call
		_readGuardiansFromEthereumToState()
		guardians := _getGuardians()

		// assert
		m.VerifyMocks()
		require.EqualValues(t, 2, _getNumberOfGuardians())
		require.EqualValues(t, g1.address, _getGuardianAtIndex(0))
		require.EqualValues(t, g2.address, _getGuardianAtIndex(1))
		require.EqualValues(t, 2, len(guardians))
		require.True(t, guardians[g1.address])
		require.True(t, guardians[g2.address])
	})
}

func TestOrbsVotingContract_processVote_readGuardiansFromEthereumWithPaging(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)

	for i := 0; i < 55; i++ {
		h.addGuardian(100)
	}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		mockGuardiansInEthereum(m, h.electionBlock, h.guardians)
		_setCurrentElectionBlockNumber(h.electionBlock)

		// call
		_readGuardiansFromEthereumToState()
		guardians := _getGuardians()

		// assert
		m.VerifyMocks()
		require.EqualValues(t, 55, _getNumberOfGuardians())
		require.EqualValues(t, 55, len(guardians))
	})
}

func TestOrbsVotingContract_processVote_collectOneGuardianStakeFromEthereum_NoStateAddr_DoesntFail(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		h.setupOrbsStateBeforeProcess()
		mockGuardianVoteInEthereum(m, h.electionBlock, [20]byte{}, [][20]byte{}, 0)

		// call
		_collectOneGuardianDataFromEthereum(0)

		// assert
		m.VerifyMocks()
	})
}

func TestOrbsVotingContract_processVote_collectGuardiansStakeFromEthereum(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	aRecentVoteBlock := h.electionBlock - 1

	var v1 = h.addValidator()
	var g1, g2 = h.addGuardian(400), h.addGuardian(600)

	g1.vote(aRecentVoteBlock, v1)
	g2.vote(aRecentVoteBlock, v1)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()
		h.setupEthereumGuardiansDataBeforeProcess(m)
		_setVotingProcessItem(0)

		// call
		i := 0
		for ; i < 2; i++ {
			_collectNextGuardiansDataFromEthereum()
		}

		// assert
		m.VerifyMocks()
		require.EqualValues(t, i, _getVotingProcessItem())
		require.EqualValues(t, 400, state.ReadUint64(_formatGuardianStakeKey(g1.address[:])))
		require.ElementsMatch(t, [][20]byte{v1.address}, _getCandidates(g1.address[:]))
		require.EqualValues(t, 600, state.ReadUint64(_formatGuardianStakeKey(g2.address[:])))
		require.ElementsMatch(t, [][20]byte{v1.address}, _getCandidates(g2.address[:]))
	})
}

func TestOrbsVotingContract_processVote_collectGuardiansDataFromEthereum_AncientVoterStakeIs0(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	anAncientVoteBlock := h.electionBlock - 2*VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS - 2

	var v1 = h.addValidator()
	var g3, g4 = h.addGuardian(100), h.addGuardian(200)

	g3.vote(anAncientVoteBlock, v1)
	g4.vote(0, v1) // fake didn't vote

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()
		h.setupEthereumGuardiansDataBeforeProcess(m)

		_setVotingProcessItem(0)

		// call
		i := 0
		for ; i < 2; i++ {
			_collectNextGuardiansDataFromEthereum()
		}

		// assert
		m.VerifyMocks()
		require.EqualValues(t, g3.address, _getGuardianAtIndex(0))
		require.EqualValues(t, 0, state.ReadUint64(_formatGuardianStakeKey(g3.address[:])))
		require.ElementsMatch(t, [][20]byte{{}}, _getCandidates(g3.address[:]))
		require.EqualValues(t, g4.address, _getGuardianAtIndex(1))
		require.EqualValues(t, 0, state.ReadUint64(_formatGuardianStakeKey(g4.address[:])))
		require.ElementsMatch(t, [][20]byte{{}}, _getCandidates(g4.address[:]))
	})
}

func TestOrbsVotingContract_processVote_collectOneDelegatorStakeFromEthereum_NoStateAddr_DoesntFail(t *testing.T) {
	electionBlock := uint64(60000)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		_setCurrentElectionBlockNumber(electionBlock)
		mockStakeInEthereum(m, electionBlock, [20]byte{}, 0)

		// call
		_collectOneDelegatorStakeFromEthereum(0)

		// assert
		m.VerifyMocks()
	})
}

func TestOrbsVotingContract_processVote_collectOneDelegatorStakeFromEthereum_IsGuardian_Stake0(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)

	var g1 = h.addGuardian(100).withIsGuardian(true)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()
		_setDelegatorAtIndex(0, g1.address[:])
		state.WriteUint64(_formatDelegatorStakeKey(g1.address[:]), 100) // previous election

		// call
		_collectOneDelegatorStakeFromEthereum(0)

		// assert
		m.VerifyMocks()
		require.EqualValues(t, 0, state.ReadUint64(_formatDelegatorStakeKey(g1.address[:])))
	})
}

func TestOrbsVotingContract_processVote_collectGuardiansStake_NoState(t *testing.T) {
	guardians := [][20]byte{{0xa1}, {0xa2}}
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		//prepare
		_setGuardians(guardians)

		// call
		guardians := _getGuardians()
		guardianStakes := _collectGuardiansStake(guardians)

		// assert
		m.VerifyMocks()
		require.Len(t, guardianStakes, 0, "should stay empty")
	})
}

func TestOrbsVotingContract_processVote_collectGuardiansStake_OnlyNumOfGuardiansInState(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()
		_setNumberOfGuardians(10)

		// call
		guardians := _getGuardians()
		guardianStakes := _collectGuardiansStake(guardians)

		// assert
		m.VerifyMocks()
		require.Len(t, guardianStakes, 0, "should stay empty")
	})
}

func TestOrbsVotingContract_processVote_collectGuardiansStake_GuardiansWithAncientVoteIgnored(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	aRecentVoteBlock := h.electionBlock - 1
	anAncientVoteBlock := h.electionBlock - 2*VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS - 2

	var v1 = h.addValidator()
	var g1, g2, g3, g4 = h.addGuardian(100), h.addGuardian(200), h.addGuardian(300), h.addGuardian(400)

	g1.vote(aRecentVoteBlock, v1)
	g2.vote(aRecentVoteBlock, v1)
	g3.vote(anAncientVoteBlock, v1)
	g4.vote(0, v1) // fake didn't vote

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()

		// call
		guardians := _getGuardians()
		guardianStakes := _collectGuardiansStake(guardians)

		// assert
		m.VerifyMocks()
		require.Len(t, guardians, 4)
		require.Len(t, guardianStakes, 2)
		_, ok := guardianStakes[g3.address]
		require.False(t, ok, "g3 should not exist ")
		_, ok = guardianStakes[g4.address]
		require.False(t, ok, "g4 should not exist ")
	})
}

func TestOrbsVotingContract_processVote_collectDelegatorStake_DelegatorIgnoredIfIsGuardian(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	aRecentVoteBlock := h.electionBlock - 1

	var g1 = h.addGuardian(100)

	g1.vote(aRecentVoteBlock, h.addValidator())

	h.addDelegator(500, g1.address)
	d2 := h.addDelegator(500, g1.address)
	h.addDelegator(500, g1.address)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()

		// call
		guardians := _getGuardians()
		guardians[d2.address] = true
		delegatorStakes := _collectDelegatorsStake(guardians)

		// assert
		m.VerifyMocks()
		require.Len(t, delegatorStakes, 2)
		_, ok := delegatorStakes[d2.address]
		require.False(t, ok, "d2 should not exist as delegator")
	})
}

func TestOrbsVotingContract_processVote_findGuardianDelegators_IgnoreSelfDelegation(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)
	h.addDelegator(500, [20]byte{})
	h.delegators[0].delegate = h.delegators[0].address

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()

		// call
		guardians := _getGuardians()
		delegatorStakes := _collectDelegatorsStake(guardians)
		guardianDelegators := _findGuardianDelegators(delegatorStakes)

		// assert
		m.VerifyMocks()
		require.Len(t, guardianDelegators, 0)
	})
}

func TestOrbsVotingContract_processVote_calculateOneGuardianVoteRecursive(t *testing.T) {
	guardian := [20]byte{0xa0}
	delegatorStakes := map[[20]byte]uint64{
		{0xb0}: 100,
		{0xb1}: 200,
		{0xb2}: 300,
		{0xb3}: 400,
	}
	tests := []struct {
		name                   string
		expect                 uint64
		relationship           map[[20]byte][][20]byte
		expectParticipantStake map[[20]byte]uint64 // will not include the guardian stake in it.
	}{
		{"simple one delegate", 200, map[[20]byte][][20]byte{{0xa0}: {{0xb1}}}, map[[20]byte]uint64{{0xb1}: 200}},
		{"simple two delegates", 600, map[[20]byte][][20]byte{{0xa0}: {{0xb1}, {0xb3}}}, map[[20]byte]uint64{{0xb3}: 400, {0xb1}: 200}},
		{"simple all delegates", 1000, map[[20]byte][][20]byte{{0xa0}: {{0xb1}, {0xb0}, {0xb2}, {0xb3}}}, map[[20]byte]uint64{{0xb3}: 400, {0xb2}: 300, {0xb1}: 200, {0xb0}: 100}},
		{"level one has another delegate", 500, map[[20]byte][][20]byte{{0xa0}: {{0xb1}}, {0xb1}: {{0xb2}}}, map[[20]byte]uint64{{0xb2}: 300, {0xb1}: 200}},
		{"simple and level one has another delegate", 600, map[[20]byte][][20]byte{{0xa0}: {{0xb0}, {0xb1}}, {0xb1}: {{0xb2}}}, map[[20]byte]uint64{{0xb2}: 300, {0xb1}: 200, {0xb0}: 100}},
		{"level one has another two delegate", 900, map[[20]byte][][20]byte{{0xa0}: {{0xb1}}, {0xb1}: {{0xb2}, {0xb3}}}, map[[20]byte]uint64{{0xb2}: 300, {0xb1}: 200, {0xb3}: 400}},
		{"level two has level one has another two delegate", 1000, map[[20]byte][][20]byte{{0xa0}: {{0xb0}}, {0xb0}: {{0xb1}}, {0xb1}: {{0xb2}, {0xb3}}}, map[[20]byte]uint64{{0xb3}: 400, {0xb2}: 300, {0xb1}: 200, {0xb0}: 100}},
	}
	for i := range tests {
		cTest := tests[i]
		t.Run(cTest.name, func(t *testing.T) {
			participant := make(map[[20]byte]uint64)
			stakes := _calculateOneGuardianVoteRecursive(guardian, cTest.relationship, delegatorStakes, participant)
			require.EqualValues(t, cTest.expect, stakes, fmt.Sprintf("%s was calculated to %d instead of %d", cTest.name, stakes, cTest.expect))
			require.EqualValues(t, len(cTest.expectParticipantStake), len(participant), "participants length not equal")
			for k, v := range participant {
				require.EqualValues(t, cTest.expectParticipantStake[k], v, "bad values")
			}
		})
	}
}

func TestOrbsVotingContract_processVote_guardiansCastVotes(t *testing.T) {
	g0, g1, g2, g3 := [20]byte{0xa0}, [20]byte{0xa1}, [20]byte{0xa2}, [20]byte{0xa3}
	delegatorStakes := map[[20]byte]uint64{
		{0xa0, 0xb0}: 100, {0xa0, 0xb1}: 200,
		{0xa1, 0xb0}: 100, {0xa1, 0xb1}: 200, {0xa1, 0xb2}: 300,
		{0xa2, 0xb0}: 100, {0xa2, 0xb1}: 200, {0xa2, 0xb2}: 300, {0xa2, 0xb3}: 400,
	}
	relationship := map[[20]byte][][20]byte{
		g0: {{0xa0, 0xb0}, {0xa0, 0xb1}},                               // 300
		g1: {{0xa1, 0xb0}, {0xa1, 0xb1}}, {0xa1, 0xb1}: {{0xa1, 0xb2}}, // 600
		g2: {{0xa2, 0xb0}}, {0xa2, 0xb0}: {{0xa2, 0xb1}}, {0xa2, 0xb1}: {{0xa2, 0xb2}, {0xa2, 0xb3}}, // 1000
	}
	v1, v2, v3, v4, v5 := [20]byte{0xc1}, [20]byte{0xc2}, [20]byte{0xc3}, [20]byte{0xc4}, [20]byte{0xc5}
	g0Vote := [][20]byte{v1, v2}
	g1Vote := [][20]byte{v3, v4, v5}
	g2Vote := [][20]byte{v1, v3, v5}
	g3Vote := make([][20]byte, 0) // voted for nonec

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCandidates(g0[:], g0Vote)
		_setCandidates(g1[:], g1Vote)
		_setCandidates(g2[:], g2Vote)
		_setCandidates(g3[:], g3Vote)
		_setCurrentElectionBlockNumber(50000)

		tests := []struct {
			name          string
			expect        map[[20]byte]uint64
			expectedTotal uint64
			guardianStake map[[20]byte]uint64
		}{
			{"simple one guardian", map[[20]byte]uint64{v1: 320, v2: 320}, 320, map[[20]byte]uint64{g0: 20}},
			{"simple two guardian", map[[20]byte]uint64{v1: 320, v2: 320, v3: 700, v4: 700, v5: 700}, 1020, map[[20]byte]uint64{g0: 20, g1: 100}},
			{"simple three guardian", map[[20]byte]uint64{v1: 1330, v2: 320, v3: 1710, v4: 700, v5: 1710}, 2030, map[[20]byte]uint64{g0: 20, g1: 100, g2: 10}},
			{"simple second guardian no delegates", map[[20]byte]uint64{v1: 320, v2: 320}, 370, map[[20]byte]uint64{g0: 20, g3: 50}},
		}
		for i := range tests {
			cTest := tests[i]
			candidatesVotes, total, _, _ := _guardiansCastVotes(cTest.guardianStake, relationship, delegatorStakes)
			require.EqualValues(t, cTest.expectedTotal, total)
			for validator, vote := range cTest.expect {
				require.EqualValues(t, vote, candidatesVotes[validator])
			}
		}
	})
}

func TestOrbsVotingContract_processVote_processValidatorsSelection(t *testing.T) {
	v1, v2, v3, v4, v5 := [20]byte{0xc1}, [20]byte{0xc2}, [20]byte{0xc3}, [20]byte{0xc4}, [20]byte{0xc5}

	tests := []struct {
		name     string
		expect   [][20]byte
		original map[[20]byte]uint64
		maxVotes uint64
	}{
		{"all pass", [][20]byte{v1, v2, v3, v4}, map[[20]byte]uint64{v1: 320, v2: 200, v3: 400, v4: 500}, 1000},
		{"one voted out", [][20]byte{v1, v3, v4}, map[[20]byte]uint64{v1: 320, v2: 701, v3: 400, v4: 699}, 1000},
		{"non valid also voted out", [][20]byte{v1, v3, v4}, map[[20]byte]uint64{v1: 320, v2: 701, v3: 400, v4: 699, v5: 400}, 1000},
	}
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setNumberOfValidators(4)
		_setValidatorEthereumAddressAtIndex(0, v1[:])
		_setValidatorEthereumAddressAtIndex(1, v2[:])
		_setValidatorEthereumAddressAtIndex(2, v3[:])
		_setValidatorEthereumAddressAtIndex(3, v4[:])
		_setCurrentElectionBlockNumber(50000)

		for i := range tests {
			cTest := tests[i]
			validCandidates := _processValidatorsSelection(cTest.original, cTest.maxVotes)
			require.Equal(t, len(cTest.expect), len(validCandidates))
			require.ElementsMatch(t, cTest.expect, validCandidates)
		}
	})
}

/***
 * driver
 */

type harness struct {
	electionBlock uint64
	blockNumber   uint64

	nextGuardianAddress      byte
	nextDelegatorAddress     byte
	nextValidatorAddress     byte
	nextValidatorOrbsAddress byte

	guardians  []*guardian
	delegators []*delegator
	validators []*validator
}

type actor struct {
	stake   int
	address [20]byte
}

type guardian struct {
	actor
	voteBlock       uint64
	votedValidators [][20]byte
	isGuardian      bool
}

func (g *guardian) withIsGuardian(isGuardian bool) *guardian {
	g.isGuardian = isGuardian
	return g
}

type delegator struct {
	actor
	delegate [20]byte
}

type validator struct {
	actor
	orbsAddress [20]byte
}

func getValidatorAddresses(validatorObjs []*validator) [][20]byte {
	addresses := make([][20]byte, 0)
	for _, v := range validatorObjs {
		addresses = append(addresses, v.address)
	}
	return addresses
}
func (g *guardian) vote(asOfBlock uint64, validators ...*validator) {
	g.voteBlock = asOfBlock
	g.votedValidators = getValidatorAddresses(validators)
}

func newHarness() *harness {
	ETHEREUM_STAKE_FACTOR = big.NewInt(int64(10000))
	VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS = 3
	VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS = 500
	ELECTION_PERIOD_LENGTH_IN_BLOCKS = 200
	MIN_ELECTED_VALIDATORS = 3
	MAX_ELECTED_VALIDATORS = 10
	return &harness{nextGuardianAddress: 0xa1, nextDelegatorAddress: 0xb1, nextValidatorAddress: 0xd1, nextValidatorOrbsAddress: 0xe1}
}

func (f *harness) addGuardian(stake int) *guardian {
	g := &guardian{actor: actor{stake: stake, address: [20]byte{f.nextGuardianAddress}}, isGuardian: true}
	f.nextGuardianAddress++
	f.guardians = append(f.guardians, g)
	return g
}

func (f *harness) addDelegator(stake int, delegate [20]byte) *delegator {
	d := &delegator{actor: actor{stake: stake, address: [20]byte{f.nextDelegatorAddress}}, delegate: delegate}
	f.nextDelegatorAddress++
	f.delegators = append(f.delegators, d)
	return d
}

func (f *harness) addValidator() *validator {
	return f.addValidatorWithStake(0)
}
func (f *harness) addValidatorWithStake(stake int) *validator {
	v := &validator{actor: actor{stake: stake, address: [20]byte{f.nextValidatorAddress}}, orbsAddress: [20]byte{f.nextValidatorOrbsAddress}}
	f.nextValidatorAddress++
	f.nextValidatorOrbsAddress++
	f.validators = append(f.validators, v)
	return v
}

func (f *harness) setupOrbsStateBeforeProcess() {
	_setCurrentElectionBlockNumber(f.electionBlock)
	f.mockDelegationsInOrbsBeforeProcess()
	f.mockGuardianInOrbsBeforeProcess()
	f.mockGuardianVotesInOrbsBeforeProcess()
	f.mockValidatorsInOrbsBeforeProcess()
}

func (f *harness) mockGuardianInOrbsBeforeProcess() {
	addresses := make([][20]byte, 0, len(f.guardians))
	for _, g := range f.guardians {
		if g.isGuardian {
			addresses = append(addresses, g.address)
		}
	}
	_setGuardians(addresses)
}

func (f *harness) mockGuardianVotesInOrbsBeforeProcess() {
	_setNumberOfGuardians(len(f.guardians))
	for i, guardian := range f.guardians {
		_setCandidates(guardian.address[:], guardian.votedValidators)
		if guardian.voteBlock != 0 && safeuint64.Add(guardian.voteBlock, VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS) > f.electionBlock {
			_setGuardianVoteBlockNumber(guardian.address[:], guardian.voteBlock)
		}
		_setGuardianStake(guardian.address[:], uint64(guardian.stake))
		_setGuardianAtIndex(i, guardian.address[:])
	}
}

func (f *harness) mockDelegationsInOrbsBeforeProcess() {
	_setNumberOfDelegators(len(f.delegators))
	for i, d := range f.delegators {
		state.WriteBytes(_formatDelegatorAgentKey(d.address[:]), d.delegate[:])
		state.WriteBytes(_formatDelegatorIterator(i), d.address[:])
	}
}

func (f *harness) mockValidatorsInOrbsBeforeProcess() {
	_setNumberOfValidators(len(f.validators))
	for i, v := range f.validators {
		state.WriteBytes(_formatValidaorIterator(i), v.address[:])
	}
}

func (f *harness) runProcessVoteNtimes(maxNumberOfRuns int) ([][20]byte, int) {
	elected := _processVotingStateMachine()
	i := 0
	if maxNumberOfRuns <= 0 {
		maxNumberOfRuns = 100
	}
	for i := 0; i < maxNumberOfRuns && elected == nil; i++ {
		elected = _processVotingStateMachine()
	}
	return elected, i
}

func (f *harness) setupEthereumStateBeforeProcess(m Mockery) {
	f.setupEthereumValidatorsBeforeProcess(m)

	mockGuardiansInEthereum(m, f.electionBlock, f.guardians)
	f.setupEthereumGuardiansDataBeforeProcess(m)

	for _, d := range f.delegators {
		mockStakeInEthereum(m, f.electionBlock, d.address, d.stake)
	}
}

func (f *harness) setupEthereumGuardiansDataBeforeProcess(m Mockery) {
	for _, a := range f.guardians {
		if a.isGuardian {
			mockGuardianVoteInEthereum(m, f.electionBlock, a.address, a.votedValidators, a.voteBlock)
			if a.voteBlock > f.electionBlock-VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS {
				mockStakeInEthereum(m, f.electionBlock, a.address, a.stake)
			}
		}
	}
}

func (f *harness) setupEthereumValidatorsBeforeProcess(m Mockery) {
	if len(f.validators) != 0 {
		validatorAddresses := make([][20]byte, len(f.validators))
		for i, a := range f.validators {
			validatorAddresses[i] = a.address
			mockStakeInEthereum(m, f.electionBlock, a.address, a.stake)
			mockValidatorOrbsAddressInEthereum(m, f.electionBlock, a.address, a.orbsAddress)
		}
		mockValidatorsInEthereum(m, f.electionBlock, validatorAddresses)
	}
}

func mockGuardianInEthereum(m Mockery, blockNumber uint64, address [20]byte, isGuardian bool) {
	m.MockEthereumCallMethodAtBlock(blockNumber, getGuardiansEthereumContractAddress(), getGuardiansAbi(), "isGuardian", func(out interface{}) {
		i, ok := out.(*bool)
		if ok {
			*i = isGuardian
		} else {
			panic(fmt.Sprintf("wrong something %s", out))
		}
	}, address)
}

func mockGuardianVoteInEthereum(m Mockery, blockNumber uint64, address [20]byte, candidates [][20]byte, voteBlockNumber uint64) {
	vote := Vote{
		ValidatorsBytes20: candidates,
		BlockNumber:       big.NewInt(int64(voteBlockNumber)),
	}
	m.MockEthereumCallMethodAtBlock(blockNumber, getVotingEthereumContractAddress(), getVotingAbi(), "getCurrentVoteBytes20", func(out interface{}) {
		i, ok := out.(*Vote)
		if ok {
			*i = vote
		} else {
			panic(fmt.Sprintf("wrong something %s", out))
		}
	}, address)
}

func mockGuardiansInEthereum(m Mockery, blockNumber uint64, guardians []*guardian) {
	addresses := make([][20]byte, 0, len(guardians))
	for _, g := range guardians {
		if g.isGuardian {
			addresses = append(addresses, g.address)
		}
	}
	m.MockEthereumCallMethodAtBlock(blockNumber, getGuardiansEthereumContractAddress(), getGuardiansAbi(), "getGuardiansBytes20", func(out interface{}) {
		ethAddresses, ok := out.(*[][20]byte)
		if ok {
			if len(addresses) > 50 {
				*ethAddresses = addresses[:50]
			} else {
				*ethAddresses = addresses
			}
		} else {
			panic(fmt.Sprintf("wrong type %s", out))
		}
	}, big.NewInt(0), big.NewInt(50))
	if len(addresses) > 50 {
		m.MockEthereumCallMethodAtBlock(blockNumber, getGuardiansEthereumContractAddress(), getGuardiansAbi(), "getGuardiansBytes20", func(out interface{}) {
			ethAddresses, ok := out.(*[][20]byte)
			if ok {
				*ethAddresses = addresses[50:]
			} else {
				panic(fmt.Sprintf("wrong type %s", out))
			}
		}, big.NewInt(50), big.NewInt(50))
	}
}

func mockValidatorsInEthereum(m Mockery, blockNumber uint64, addresses [][20]byte) {
	m.MockEthereumCallMethodAtBlock(blockNumber, getValidatorsEthereumContractAddress(), getValidatorsAbi(), "getValidatorsBytes20", func(out interface{}) {
		ethAddresses, ok := out.(*[][20]byte)
		if ok {
			*ethAddresses = addresses
		} else {
			panic(fmt.Sprintf("wrong type %s", out))
		}
	})
}

func mockValidatorOrbsAddressInEthereum(m Mockery, blockNumber uint64, validatorAddress [20]byte, orbsValidatorAddress [20]byte) {
	m.MockEthereumCallMethodAtBlock(blockNumber, getValidatorsRegistryEthereumContractAddress(), getValidatorsRegistryAbi(),
		"getOrbsAddress", func(out interface{}) {
			orbsAddress, ok := out.(*[20]byte)
			if ok {
				*orbsAddress = orbsValidatorAddress
			} else {
				panic(fmt.Sprintf("wrong something %s", out))
			}
		}, validatorAddress)
}

func mockStakeInEthereum(m Mockery, blockNumber uint64, address [20]byte, stake int) {
	stakeValue := big.NewInt(int64(stake))
	stakeValue = stakeValue.Mul(stakeValue, ETHEREUM_STAKE_FACTOR)
	m.MockEthereumCallMethodAtBlock(blockNumber, getTokenEthereumContractAddress(), getTokenAbi(), "balanceOf", func(out interface{}) {
		i, ok := out.(**big.Int)
		if ok {
			*i = stakeValue
		} else {
			panic(fmt.Sprintf("wrong something %s", out))
		}
	}, address)
}

/***
 * helpers
 */
func setTimingInMirror(m Mockery) {
	election := uint64(150)
	setTiming(m, election, int(election+VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS)-2)
}
func setTiming(m Mockery, electionBlock uint64, currentBlock int) {
	m.MockEthereumGetBlockNumber(currentBlock)
	_setCurrentElectionBlockNumber(electionBlock)
}
