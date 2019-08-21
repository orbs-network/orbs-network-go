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

func TestOrbsVotingContract_annualFactorize(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		require.EqualValues(t, 75, _annualFactorize(888000))
		switchToTimeBasedElections()
		require.EqualValues(t, 72, _annualFactorize(888000))
	})
}

func TestOrbsVotingContract_processRewards_getValidatorStakes(t *testing.T) {
	validators := [][20]byte{{0x01}, {0x02}, {0x03}}
	stakes := []uint64{100, 200}
	stakesReal := []uint64{100, 200, 0}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
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
	totalVotes := uint64(820000)
	p1, p2, p3, p4, p5 := [20]byte{0xa0}, [20]byte{0xb1}, [20]byte{0xc1}, [20]byte{0xd1}, [20]byte{0xe1}
	participants := [][20]byte{p1, p2, p3, p4, p5}
	participantStakes := map[[20]byte]uint64{p1: 100000, p2: 50000, p3: 0, p4: 10000, p5: 40000}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_processRewardsParticipants(totalVotes, participants, participantStakes)

		// assert
		require.EqualValues(t, 34, getCumulativeParticipationReward(p2[:]))
		require.EqualValues(t, 6, getCumulativeParticipationReward(p4[:]))
		require.EqualValues(t, 68, getCumulativeParticipationReward(p1[:]))
		require.EqualValues(t, 0, getCumulativeParticipationReward(p3[:]))
		require.EqualValues(t, 27, getCumulativeParticipationReward(p5[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsParticipants_TotalAboveMax(t *testing.T) {
	totalVotes := uint64(800000000)
	h := newRewardHarness()
	p1, p2, p3 := h.addStakeActor(1000000), h.addStakeActor(200000000), h.addStakeActor(0)
	participants := [][20]byte{p1.address, p2.address, p3.address}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_processRewardsParticipants(totalVotes, participants, h.getAllStakes())

		// assert
		max := ELECTION_PARTICIPATION_MAX_REWARD * 100 / ANNUAL_TO_ELECTION_FACTOR_BLOCKBASED
		require.EqualValues(t, max/4, getCumulativeParticipationReward(p2.address[:]))
		require.EqualValues(t, max/800, getCumulativeParticipationReward(p1.address[:]))
		require.EqualValues(t, 0, getCumulativeParticipationReward(p3.address[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsGuardians_SmallNumberOfGuardians_SmallTotal(t *testing.T) {
	totalVotes := uint64(820000)
	p1, p2, p3, p4, p5 := [20]byte{0xa0}, [20]byte{0xb1}, [20]byte{0xc1}, [20]byte{0xd1}, [20]byte{0xe1}
	guardiansAccumulatedStakes := map[[20]byte]uint64{p1: 540000, p2: 250000, p3: 0, p4: 10000, p5: 20000}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_processRewardsGuardians(totalVotes, guardiansAccumulatedStakes)

		// assert
		require.EqualValues(t, 213, getCumulativeGuardianExcellenceReward(p2[:]))
		require.EqualValues(t, 8, getCumulativeGuardianExcellenceReward(p4[:]))
		require.EqualValues(t, 460, getCumulativeGuardianExcellenceReward(p1[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(p3[:]))
		require.EqualValues(t, 17, getCumulativeGuardianExcellenceReward(p5[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsGuardians_SmallNumberOfGuardians_LargeTotal(t *testing.T) {
	totalVotes := uint64(500000000)
	p1, p2, p3 := [20]byte{0xa0}, [20]byte{0xb1}, [20]byte{0xc1}
	guardiansAccumulatedStakes := map[[20]byte]uint64{p1: 400000000, p2: 100000000, p3: 0}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_processRewardsGuardians(totalVotes, guardiansAccumulatedStakes)

		// assert
		max := ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD * 100 / ANNUAL_TO_ELECTION_FACTOR_BLOCKBASED
		require.EqualValues(t, max*4/5, getCumulativeGuardianExcellenceReward(p1[:]))
		require.EqualValues(t, max/5, getCumulativeGuardianExcellenceReward(p2[:]))
		require.EqualValues(t, 0, getCumulativeParticipationReward(p3[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsGuardians_LargeNumberOfGuardians_Exactly10Top__SmallTotal(t *testing.T) {
	h := newRewardHarness()
	for i := 0; i < ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER; i++ {
		h.addStakeActor(100000*i + 200000)
	}
	p1, p2, p3 := h.addStakeActor(2200000), h.addStakeActor(1200000), h.addStakeActor(100000)
	calculatedTotal := 0
	calcualtedTotalRewardFromStake := uint64(0)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_processRewardsGuardians(0, h.getAllStakes())

		// assert
		for i := 2; i < 12; i++ {
			calculatedTotal += h.getActor(i).stake
		}
		for i := 0; i < h.getNumActors(); i++ {
			calcualtedTotalRewardFromStake += getCumulativeGuardianExcellenceReward(h.getActor(i).address[:])
		}

		require.EqualValues(t, 8013, calcualtedTotalRewardFromStake)
		require.EqualValues(t, 1876, getCumulativeGuardianExcellenceReward(p1.address[:]))
		require.EqualValues(t, 1023, getCumulativeGuardianExcellenceReward(p2.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(p3.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(0).address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(1).address[:]))
		require.EqualValues(t, 341, getCumulativeGuardianExcellenceReward(h.getActor(2).address[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsGuardians_LargeNumberOfGuardians_MoreThan10Top_SmallTotal(t *testing.T) {
	h := newRewardHarness()
	for i := 0; i < ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER; i++ {
		h.addStakeActor(100000*i + 200000)
	}
	p1, p2, p3, p4 := h.addStakeActor(2200000), h.addStakeActor(1200000), h.addStakeActor(100000), h.addStakeActor(400000)
	calculatedTotal := 0
	calcualtedTotalRewardFromStake := uint64(0)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_processRewardsGuardians(0, h.getAllStakes())

		// assert
		for i := 2; i < 12; i++ {
			calculatedTotal += h.getActor(i).stake
		}
		for i := 0; i < h.getNumActors(); i++ {
			calcualtedTotalRewardFromStake += getCumulativeGuardianExcellenceReward(h.getActor(i).address[:])
		}

		require.EqualValues(t, 220, len(getExcellenceProgramGuardians()))
		require.EqualValues(t, 8354, calcualtedTotalRewardFromStake)
		require.EqualValues(t, 1876, getCumulativeGuardianExcellenceReward(p1.address[:]))
		require.EqualValues(t, 1023, getCumulativeGuardianExcellenceReward(p2.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(p3.address[:]))
		require.EqualValues(t, 341, getCumulativeGuardianExcellenceReward(p4.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(0).address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(1).address[:]))
		require.EqualValues(t, 341, getCumulativeGuardianExcellenceReward(h.getActor(2).address[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsParticipants_LargeNumberOfGuardians_LargeTotal(t *testing.T) {
	h := newRewardHarness()
	for i := 0; i < ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER; i++ {
		h.addStakeActor(10000000*i + 20000000)
	}
	p1, p2, p3 := h.addStakeActor(220000000), h.addStakeActor(120000000), h.addStakeActor(10000000)
	calculatedTotal := uint64(0)
	calcualtedTotalRewardsFromStake := uint64(0)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		_processRewardsGuardians(0, h.getAllStakes())

		// assert
		for i := 2; i < 12; i++ {
			calculatedTotal += uint64(h.getActor(i).stake)
		}
		for i := 0; i < h.getNumActors(); i++ {
			calcualtedTotalRewardsFromStake += getCumulativeGuardianExcellenceReward(h.getActor(i).address[:])
		}
		max := ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD * 100 / ANNUAL_TO_ELECTION_FACTOR_BLOCKBASED
		require.True(t, max-calcualtedTotalRewardsFromStake < 10) // rounding error
		require.EqualValues(t, 79857, getCumulativeGuardianExcellenceReward(p1.address[:]))
		require.EqualValues(t, 43558, getCumulativeGuardianExcellenceReward(p2.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(p3.address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(0).address[:]))
		require.EqualValues(t, 0, getCumulativeGuardianExcellenceReward(h.getActor(1).address[:]))
		require.EqualValues(t, 14519, getCumulativeGuardianExcellenceReward(h.getActor(2).address[:]))
	})
}

func TestOrbsVotingContract_processRewards_processRewardsValidators(t *testing.T) {
	p1, p2, p3, p4, p5 := [20]byte{0xa1}, [20]byte{0xb1}, [20]byte{0xc1}, [20]byte{0xd1}, [20]byte{0xe1}
	validatorStakes := map[[20]byte]uint64{p1: 1000000, p2: 500000, p3: 0, p4: 100000, p5: 400000}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setNumberOfValidators(len(validatorStakes))
		_setValidatorEthereumAddressAtIndex(0, p1[:])
		_setValidatorStake(p1[:], uint64(1000000))
		_setValidatorEthereumAddressAtIndex(1, p2[:])
		_setValidatorStake(p2[:], uint64(500000))
		_setValidatorEthereumAddressAtIndex(2, p3[:])
		_setValidatorStake(p3[:], uint64(0))
		_setValidatorEthereumAddressAtIndex(3, p4[:])
		_setValidatorStake(p4[:], uint64(100000))
		_setValidatorEthereumAddressAtIndex(4, p5[:])
		_setValidatorStake(p5[:], uint64(400000))

		// call
		_processRewardsValidators([][20]byte{p1, p2, p3, p4})

		// assert
		electionValidatorIntroduction := ELECTION_VALIDATOR_INTRODUCTION_REWARD * 100 / ANNUAL_TO_ELECTION_FACTOR_BLOCKBASED
		require.EqualValues(t, electionValidatorIntroduction+341, getCumulativeValidatorReward(p1[:]))
		require.EqualValues(t, electionValidatorIntroduction+170, getCumulativeValidatorReward(p2[:]))
		require.EqualValues(t, electionValidatorIntroduction+0, getCumulativeValidatorReward(p3[:]))
		require.EqualValues(t, electionValidatorIntroduction+34, getCumulativeValidatorReward(p4[:]))
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
		{"participant under max", 5, 493150, 8200, 8},
		{"participant over max", 4206, 493150, 62000000, 8},
	}
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
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
