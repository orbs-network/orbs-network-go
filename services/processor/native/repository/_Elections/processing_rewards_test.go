// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"fmt"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsVotingContract_processRewards_getValidatorStakes(t *testing.T) {
	validators := [][20]byte{{0x01}, {0x02}, {0x03}}
	stakes := []uint64{100, 200}
	stakesReal := []uint64{100, 200, 0}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		_setCurrentElectionBlockNumber(5000)
		_setNumberOfValidators(len(validators))
		for i := 0; i < len(validators); i++ {
			_setValidatorEthereumAddressAtIndex(i, validators[i][:])
		}
		for i := 0; i < len(stakes); i++ {
			_setValidatorStake(validators[i][:], stakes[i])
		}

		// call
		vtoS := _getValidatorsStake()

		// assert
		require.EqualValues(t, len(validators), len(vtoS))
		for i := 0; i < _getNumberOfValidators(); i++ {
			require.EqualValues(t, validators[i], _getValidatorEthereumAddressAtIndex(i))
			require.EqualValues(t, stakesReal[i], getValidatorStake(validators[i][:]))
		}
	})
}

func TestOrbsVotingContract_processRewards_processRewardsParticipants(t *testing.T) {
	totalVotes := uint64(8200)
	p1, p2, p3, p4, p5 := [20]byte{0xa0}, [20]byte{0xb1}, [20]byte{0xc1}, [20]byte{0xd1}, [20]byte{0xe1}
	participantStakes := map[[20]byte]uint64{p1: 1000, p2: 500, p3: 0, p4: 100, p5: 400}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(5000)

		// call
		_processRewardsParticipants(totalVotes, participantStakes)

		// assert
		require.EqualValues(t, 40, getCumulativeParticipationReward(p2[:]))
		require.EqualValues(t, 8, getCumulativeParticipationReward(p4[:]))
		require.EqualValues(t, 80, getCumulativeParticipationReward(p1[:]))
		require.EqualValues(t, 0, getCumulativeParticipationReward(p3[:]))
		require.EqualValues(t, 32, getCumulativeParticipationReward(p5[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsParticipants_TotalAboveMax(t *testing.T) {
	totalVotes := uint64(62000000)
	h := newRewardHarness()
	p1, p2, p3 := h.addStakeActor(1000000), h.addStakeActor(31000000), h.addStakeActor(0)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(5000)

		// call
		_processRewardsParticipants(totalVotes, h.getAllStakes())

		// assert
		require.EqualValues(t, ELECTION_PARTICIPATION_MAX_REWARD/62, getCumulativeParticipationReward(p1.address[:]))
		require.EqualValues(t, ELECTION_PARTICIPATION_MAX_REWARD/2, getCumulativeParticipationReward(p2.address[:]))
		require.EqualValues(t, 0, getCumulativeParticipationReward(p3.address[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsParticipants_SmallNumberOfGuardians_SmallTotal(t *testing.T) {
	totalVotes := uint64(8200)
	p1, p2, p3, p4, p5 := [20]byte{0xa0}, [20]byte{0xb1}, [20]byte{0xc1}, [20]byte{0xd1}, [20]byte{0xe1}
	guardiansAccumulatedStakes := map[[20]byte]uint64{p1: 5400, p2: 2500, p3: 0, p4: 100, p5: 200}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(5000)

		// call
		_processRewardsGuardians(totalVotes, guardiansAccumulatedStakes)

		// assert
		require.EqualValues(t, 250, getCumulativeGuardianExcellenceReward(p2[:]))
		require.EqualValues(t, 10, getCumulativeGuardianExcellenceReward(p4[:]))
		require.EqualValues(t, 540, getCumulativeGuardianExcellenceReward(p1[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(p3[:]))
		require.EqualValues(t, 20, getCumulativeGuardianExcellenceReward(p5[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsParticipants_SmallNumberOfGuardians_LargeTotal(t *testing.T) {
	totalVotes := uint64(50000000)
	p1, p2, p3 := [20]byte{0xa0}, [20]byte{0xb1}, [20]byte{0xc1}
	guardiansAccumulatedStakes := map[[20]byte]uint64{p1: 25000000, p2: 1000000, p3: 0}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(5000)

		// call
		_processRewardsGuardians(totalVotes, guardiansAccumulatedStakes)

		// assert
		require.EqualValues(t, ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD/50, getCumulativeGuardianExcellenceReward(p2[:]))
		require.EqualValues(t, ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD/2, getCumulativeGuardianExcellenceReward(p1[:]))
		require.EqualValues(t, 0, getCumulativeParticipationReward(p3[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsParticipants_LargeNumberOfGuardians_SmallTotal(t *testing.T) {
	h := newRewardHarness()
	for i := 0; i < ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER; i++ {
		h.addStakeActor(1000*i + 2000)
	}
	p1, p2, p3 := h.addStakeActor(22000), h.addStakeActor(12000), h.addStakeActor(1000)
	calculatedTotal := 0
	calcualtedTotalRewardFromStake := uint64(0)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(5000)

		// call
		_processRewardsGuardians(0, h.getAllStakes())

		// assert
		for i := 2; i < 12; i++ {
			calculatedTotal += h.getActor(i).stake
		}
		calculatedTotalReward := calculatedTotal * int(ELECTION_GUARDIAN_EXCELLENCE_MAX_STAKE_REWARD_PERCENT) / 100
		for i := 0; i < h.getNumActors(); i++ {
			calcualtedTotalRewardFromStake += getCumulativeGuardianExcellenceReward(h.getActor(i).address[:])
		}

		require.EqualValues(t, calculatedTotalReward, calcualtedTotalRewardFromStake)
		require.EqualValues(t, ELECTION_GUARDIAN_EXCELLENCE_MAX_STAKE_REWARD_PERCENT, calculatedTotal/calculatedTotalReward)
		require.EqualValues(t, 2200, getCumulativeGuardianExcellenceReward(p1.address[:]))
		require.EqualValues(t, 1200, getCumulativeGuardianExcellenceReward(p2.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(p3.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(0).address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(1).address[:]))
		require.EqualValues(t, 400, getCumulativeGuardianExcellenceReward(h.getActor(2).address[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsParticipants_LargeNumberOfGuardians_LargeTotal(t *testing.T) {
	h := newRewardHarness()
	for i := 0; i < ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER; i++ {
		h.addStakeActor(1000000*i + 2000000)
	}
	p1, p2, p3 := h.addStakeActor(22000000), h.addStakeActor(12000000), h.addStakeActor(1000000)
	calculatedTotal := uint64(0)
	calcualtedTotalRewardsFromStake := uint64(0)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(5000)

		// call
		_processRewardsGuardians(0, h.getAllStakes())

		// assert
		for i := 2; i < 12; i++ {
			calculatedTotal += uint64(h.getActor(i).stake)
		}
		for i := 0; i < h.getNumActors(); i++ {
			calcualtedTotalRewardsFromStake += getCumulativeGuardianExcellenceReward(h.getActor(i).address[:])
		}
		require.True(t, ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD-calcualtedTotalRewardsFromStake < 10) // rounding error
		require.EqualValues(t, ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD*22000000/calculatedTotal, getCumulativeGuardianExcellenceReward(p1.address[:]))
		require.EqualValues(t, ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD*12000000/calculatedTotal, getCumulativeGuardianExcellenceReward(p2.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(p3.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(0).address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(1).address[:]))
		require.EqualValues(t, ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD*4000000/calculatedTotal, getCumulativeGuardianExcellenceReward(h.getActor(2).address[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsValidators(t *testing.T) {
	p1, p2, p3, p4, p5 := [20]byte{0xa1}, [20]byte{0xb1}, [20]byte{0xc1}, [20]byte{0xd1}, [20]byte{0xe1}
	validatorStakes := map[[20]byte]uint64{p1: 1000, p2: 500, p3: 0, p4: 100, p5: 400}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setNumberOfValidators(len(validatorStakes))
		_setValidatorEthereumAddressAtIndex(0, p1[:])
		_setValidatorStake(p1[:], uint64(1000))
		_setValidatorEthereumAddressAtIndex(1, p2[:])
		_setValidatorStake(p2[:], uint64(500))
		_setValidatorEthereumAddressAtIndex(2, p3[:])
		_setValidatorStake(p3[:], uint64(0))
		_setValidatorEthereumAddressAtIndex(3, p4[:])
		_setValidatorStake(p4[:], uint64(100))
		_setValidatorEthereumAddressAtIndex(4, p5[:])
		_setValidatorStake(p5[:], uint64(400))

		// call
		_processRewardsValidators([][20]byte{p1, p2, p3, p4})

		// assert
		require.EqualValues(t, ELECTION_VALIDATOR_INTRODUCTION_MAX_REWARD+40, getCumulativeValidatorReward(p1[:]))
		require.EqualValues(t, ELECTION_VALIDATOR_INTRODUCTION_MAX_REWARD+20, getCumulativeValidatorReward(p2[:]))
		require.EqualValues(t, ELECTION_VALIDATOR_INTRODUCTION_MAX_REWARD+0, getCumulativeValidatorReward(p3[:]))
		require.EqualValues(t, ELECTION_VALIDATOR_INTRODUCTION_MAX_REWARD+4, getCumulativeValidatorReward(p4[:]))
		require.EqualValues(t, 0, getCumulativeValidatorReward(p5[:]))
	})
}

func TestOrbsVotingContract_processRewards_maxRewardForGroup(t *testing.T) {
	tests := []struct {
		name    string
		expect  uint64
		max     uint64
		total   uint64
		percent uint64
	}{
		{"participant under max", 656, 493150, 8200, 8},
		{"participant over max", 493150, 493150, 62000000, 8},
	}
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(5000)
		for i := range tests {
			cTest := tests[i]
			reward := _maxRewardForGroup(cTest.max, cTest.total, cTest.percent)
			require.EqualValues(t, cTest.expect, reward, fmt.Sprintf("%s was calculated to %d instead of %d", cTest.name, reward, cTest.expect))
		}
	})
}

type rewardHarness struct {
	nextAddress byte

	actors []*actor
	stakes map[[20]byte]uint64
}

func newRewardHarness() *rewardHarness {
	return &rewardHarness{nextAddress: 0xa1, stakes: make(map[[20]byte]uint64)}
}

func (f *rewardHarness) addStakeActor(stake int) *actor {
	a := &actor{stake: stake, address: [20]byte{f.nextAddress}}
	f.nextAddress++
	f.actors = append(f.actors, a)
	f.stakes[a.address] = uint64(stake)
	return a
}

func (f *rewardHarness) getActor(i int) *actor {
	return f.actors[i]
}

func (f *rewardHarness) getNumActors() int {
	return len(f.actors)
}

func (f *rewardHarness) getAllStakes() map[[20]byte]uint64 {
	return f.stakes
}
