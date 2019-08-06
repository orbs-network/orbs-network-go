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
)

func _firstElectionFixRewards() {
	key := []byte("_fix_rewards_first_election_")
	if state.ReadUint32(key) == 0 {
		guardians := _getGuardians()
		for guardian, _ := range guardians {
			reward := getCumulativeGuardianExcellenceReward(guardian[:])
			if reward != 0 {
				state.Clear(_formatCumulativeGuardianExcellenceReward(guardian[:]))
				fmt.Printf("elections %10d rewards fix: clear guardian %x, orig reward %d\n", getEffectiveElectionBlockNumber(), guardian, reward)
				reward = safeuint64.Div(safeuint64.Mul(reward, 1000), 2229)
				_addCumulativeGuardianExcellenceReward(guardian[:], reward)
				fmt.Printf("elections %10d rewards fix: guardian %x reward %d\n", getEffectiveElectionBlockNumber(), guardian, reward)
			}
			reward = getCumulativeParticipationReward(guardian[:])
			if reward != 0 {
				state.Clear(_formatCumulativeParticipationReward(guardian[:]))
				fmt.Printf("elections %10d rewards fix: clear guardian participant %x orig reward %d\n", getEffectiveElectionBlockNumber(), guardian, reward)
				reward = safeuint64.Div(safeuint64.Mul(reward, 1000), 4179)
				_addCumulativeParticipationReward(guardian[:], reward)
				fmt.Printf("elections %10d rewards fix: guardian participant %x reward %d\n", getEffectiveElectionBlockNumber(), guardian, reward)
			}

		}

		electionValidatorIntroduction := safeuint64.Div(safeuint64.Mul(ELECTION_VALIDATOR_INTRODUCTION_REWARD, 100), ANNUAL_TO_ELECTION_FACTOR_BLOCKBASED)
		validators := _getValidators()
		for _, validator := range validators {
			reward := getCumulativeValidatorReward(validator[:])
			if reward != 0 {
				state.Clear(_formatCumulativeValidatorReward(validator[:]))
				fmt.Printf("elections %10d rewards fix: clear validator %x orig reward %d\n", getEffectiveElectionBlockNumber(), validator, reward)
				reward = safeuint64.Sub(reward, 8423)
				if reward != 0 {
					reward = safeuint64.Div(safeuint64.Mul(reward, 100), 11723)
				}
				reward = safeuint64.Add(reward, electionValidatorIntroduction)
				_addCumulativeValidatorReward(validator[:], reward)
				fmt.Printf("elections %10d rewards fix: validator %x reward %d\n", getEffectiveElectionBlockNumber(), validator, reward)
			}
		}
		state.WriteUint32(key, 1)
	} else {
		panic(fmt.Sprintf("cannot fix first election rewards anymore"))
	}
}

func _firstElectionFixRewardsDelegator(delegator []byte) {
	key := []byte(fmt.Sprintf("_fix_rewards_first_election_%s", hex.EncodeToString(delegator)))
	if state.ReadUint32(key) == 0 && getNumberOfElections() == 1 {
		reward := getCumulativeParticipationReward(delegator)
		if reward != 0 {
			state.Clear(_formatCumulativeParticipationReward(delegator))
			fmt.Printf("elections %10d rewards fix: clear participant %x original %d\n", getEffectiveElectionBlockNumber(), delegator, reward)
			reward = safeuint64.Div(safeuint64.Mul(reward, 1000), 4179)
			_addCumulativeParticipationReward(delegator[:], reward)
			fmt.Printf("elections %10d rewards fix: participant %x reward %d\n", getEffectiveElectionBlockNumber(), delegator, reward)
		}
		state.WriteUint32(key, 1)
	} else {
		panic(fmt.Sprintf("cannot fix first election rewards anymore"))
	}
}
