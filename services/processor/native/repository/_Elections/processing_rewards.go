// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/safemath/safeuint64"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"sort"
)

/***
 * Rewards
 */
var ELECTION_PARTICIPATION_MAX_REWARD = uint64(505328) // 60M / number of elections per year
var ELECTION_PARTICIPATION_MAX_STAKE_REWARD_PERCENT = uint64(8)
var ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD = uint64(336885) // 40M / number of elections per year
var ELECTION_GUARDIAN_EXCELLENCE_MAX_STAKE_REWARD_PERCENT = uint64(10)
var ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER = 10
var ELECTION_VALIDATOR_INTRODUCTION_MAX_REWARD = uint64(8423) // 1M / number of elections per year
var ELECTION_VALIDATOR_MAX_STAKE_REWARD_PERCENT = uint64(4)

func _processRewards(totalVotes uint64, elected [][20]byte, participantStakes map[[20]byte]uint64, guardiansAccumulatedStake map[[20]byte]uint64) {
	_processRewardsParticipants(totalVotes, participantStakes)
	_processRewardsGuardians(totalVotes, guardiansAccumulatedStake)
	_processRewardsValidators(elected)
}

func _processRewardsParticipants(totalVotes uint64, participantStakes map[[20]byte]uint64) {
	totalReward := _maxRewardForGroup(ELECTION_PARTICIPATION_MAX_REWARD, totalVotes, ELECTION_PARTICIPATION_MAX_STAKE_REWARD_PERCENT)
	fmt.Printf("elections %10d rewards: participants total reward is %d \n", getCurrentElectionBlockNumber(), totalReward)
	for participant, stake := range participantStakes {
		reward := safeuint64.Div(safeuint64.Mul(stake, totalReward), totalVotes)
		fmt.Printf("elections %10d rewards: participant %x, stake %d adding %d\n", getCurrentElectionBlockNumber(), participant, stake, reward)
		_addCumulativeParticipationReward(participant[:], reward)
	}
}

func _processRewardsGuardians(totalVotes uint64, guardiansAccumulatedStake map[[20]byte]uint64) {
	if len(guardiansAccumulatedStake) > ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER {
		fmt.Printf("elections %10d rewards: there are %d guardians with total reward is %d - choosing %d top guardians\n",
			getCurrentElectionBlockNumber(), len(guardiansAccumulatedStake), totalVotes, ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER)
		guardiansAccumulatedStake, totalVotes = _getTopGuardians(guardiansAccumulatedStake)
		fmt.Printf("elections %10d rewards: top %d guardians with total vote is now %d \n", getCurrentElectionBlockNumber(), len(guardiansAccumulatedStake), totalVotes)
	}

	_setExcellenceProgramGuardians(guardiansAccumulatedStake)
	totalReward := _maxRewardForGroup(ELECTION_GUARDIAN_EXCELLENCE_MAX_REWARD, totalVotes, ELECTION_GUARDIAN_EXCELLENCE_MAX_STAKE_REWARD_PERCENT)
	fmt.Printf("elections %10d rewards: guardians total reward is %d \n", getCurrentElectionBlockNumber(), totalReward)
	for guardian, stake := range guardiansAccumulatedStake {
		reward := safeuint64.Div(safeuint64.Mul(stake, totalReward), totalVotes)
		fmt.Printf("elections %10d rewards: guardian %x, stake %d adding %d\n", getCurrentElectionBlockNumber(), guardian, stake, reward)
		_addCumulativeGuardianExcellenceReward(guardian[:], reward)
	}
}

func _processRewardsValidators(elected [][20]byte) {
	fmt.Printf("elections %10d rewards: validadator introduction reward %d\n", getCurrentElectionBlockNumber(), ELECTION_VALIDATOR_INTRODUCTION_MAX_REWARD)
	validatorsStake := _getValidatorsStake()
	for _, elected := range elected {
		stake := validatorsStake[elected]
		reward := safeuint64.Add(ELECTION_VALIDATOR_INTRODUCTION_MAX_REWARD, safeuint64.Div(safeuint64.Mul(stake, ELECTION_VALIDATOR_MAX_STAKE_REWARD_PERCENT), 100))
		fmt.Printf("elections %10d rewards: validator %x, stake %d adding %d\n", getCurrentElectionBlockNumber(), elected, stake, reward)
		_addCumulativeValidatorReward(elected[:], reward)
	}
}

func _getValidatorsStake() (validatorsStake map[[20]byte]uint64) {
	numOfValidators := _getNumberOfValidators()
	validatorsStake = make(map[[20]byte]uint64, numOfValidators)
	for i := 0; i < numOfValidators; i++ {
		validator := _getValidatorEthereumAddressAtIndex(i)
		stake := getValidatorStake(validator[:])
		validatorsStake[validator] = stake
		fmt.Printf("elections %10d rewards: validator %x, stake %d\n", getCurrentElectionBlockNumber(), validator, stake)
	}
	return
}

func _maxRewardForGroup(upperMaximum, totalVotes, percent uint64) uint64 {
	calcMaximum := safeuint64.Div(safeuint64.Mul(totalVotes, percent), 100)
	fmt.Printf("elections %10d rewards: uppperMax %d vs. %d = totalVotes %d * percent %d\n", getCurrentElectionBlockNumber(), upperMaximum, calcMaximum, totalVotes, percent)
	if calcMaximum < upperMaximum {
		return calcMaximum
	}
	return upperMaximum
}

func _formatCumulativeParticipationReward(delegator []byte) []byte {
	return []byte(fmt.Sprintf("Participant_CumReward_%s", hex.EncodeToString(delegator)))
}

func getCumulativeParticipationReward(delegator []byte) uint64 {
	return state.ReadUint64(_formatCumulativeParticipationReward(delegator))
}

func _addCumulativeParticipationReward(delegator []byte, reward uint64) {
	_addCumulativeReward(_formatCumulativeParticipationReward(delegator), reward)
}

func _formatCumulativeGuardianExcellenceReward(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_CumReward_%s", hex.EncodeToString(guardian)))
}

func getCumulativeGuardianExcellenceReward(guardian []byte) uint64 {
	return state.ReadUint64(_formatCumulativeGuardianExcellenceReward(guardian))
}

func _addCumulativeGuardianExcellenceReward(guardian []byte, reward uint64) {
	_addCumulativeReward(_formatCumulativeGuardianExcellenceReward(guardian), reward)
}

func _formatCumulativeValidatorReward(validator []byte) []byte {
	return []byte(fmt.Sprintf("Vaidator_CumReward_%s", hex.EncodeToString(validator)))
}

func getCumulativeValidatorReward(validator []byte) uint64 {
	return state.ReadUint64(_formatCumulativeValidatorReward(validator))
}

func _addCumulativeValidatorReward(validator []byte, reward uint64) {
	_addCumulativeReward(_formatCumulativeValidatorReward(validator), reward)
}

func _addCumulativeReward(key []byte, reward uint64) {
	sumReward := safeuint64.Add(state.ReadUint64(key), reward)
	state.WriteUint64(key, sumReward)
}

func _formatExcellenceProgramGuardians() []byte {
	return []byte("Excellence_Program_Guardians")
}

func getExcellenceProgramGuardians() []byte {
	return state.ReadBytes(_formatExcellenceProgramGuardians())
}

func _setExcellenceProgramGuardians(guardians map[[20]byte]uint64) {
	guardiansForSave := make([]byte, 0, len(guardians)*20)
	for guardianAddr := range guardians {
		guardiansForSave = append(guardiansForSave, guardianAddr[:]...)
	}
	state.WriteBytes(_formatExcellenceProgramGuardians(), guardiansForSave)
}

/***
 * Rewards: Sort top guardians using sort.Interface
 */
func _getTopGuardians(guardiansAccumulatedStake map[[20]byte]uint64) (topGuardiansStake map[[20]byte]uint64, totalVotes uint64) {
	totalVotes = uint64(0)
	topGuardiansStake = make(map[[20]byte]uint64)

	guardianList := make(guardianArray, 0, len(guardiansAccumulatedStake))
	for guardian, vote := range guardiansAccumulatedStake {
		guardianList = append(guardianList, &guardianVote{guardian, vote})
	}
	sort.Sort(guardianList)

	for i := 0; i < ELECTION_GUARDIAN_EXCELLENCE_MAX_NUMBER; i++ {
		fmt.Printf("elections %10d rewards: top guardian %x, has %d votes\n", _getCurrentElectionBlockNumber(), guardianList[i].guardian, guardianList[i].vote)
		totalVotes = safeuint64.Add(totalVotes, guardianList[i].vote)
		topGuardiansStake[guardianList[i].guardian] = guardianList[i].vote
	}
	return
}

type guardianVote struct {
	guardian [20]byte
	vote     uint64
}
type guardianArray []*guardianVote

func (s guardianArray) Len() int {
	return len(s)
}

func (s guardianArray) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s guardianArray) Less(i, j int) bool {
	return s[i].vote > s[j].vote
}
