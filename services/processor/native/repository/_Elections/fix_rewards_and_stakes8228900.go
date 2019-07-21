// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

/**

These two function fix the state for rewards and delegation and can be called only once (and between election 7928900 and 7948900).

Fixed a bug in the reward calculation contract that incorrectly counted a small number of delegations.
This issue affected 29 participating addresses, to the total sum of 89,668 ORBS.

_fixRewardsDrift8208900 : fixes the 20 delegators and 9 guardians rewards according to addresses
_fixDelegatorState8208900 : remove the double delegation pointer from the delegators' list in state.

*/

var participantsFixDoubleDelegationReward8208900 = []struct {
	address string
	reward  int
}{
	{"f7aba9b064a12330a00eafaa930e2fe8e76e65f0", 48}, // instead of 72
}

var guardiansFixDoubleDelegationReward8208900 = []struct {
	address string
	reward  int
}{
	{"63AEf7616882F488BCa97361d1c24F05B4657ae5", 2051117}, // instead of 2051111
	{"f7ae622C77D0580f02Bcb2f92380d61e3F6e466c", 2377419}, // instead of 2377415
	{"8287928a809346dF4Cd53A096025a1136F7C4fF5", 998938},  // instead of 998966
	{"C82eC0F3c834d337c51c9375a1C0A65Ce7aaDaec", 1422816}, // instead of 1422813
	{"F058cCFB2324310C33E8FD9a1ddA8E99C8bEdA59", 886439},  // instead of 886437
	{"9afc8EF233e2793B2b90Ca5d70cA2e7098013142", 837312},  // instead of 837310
	{"0874BC1383958e2475dF73dC68C4F09658E23777", 397564},  // instead of 397561
}

func _fixRewardsDrift8208900() {
	key := []byte("_fixRewards8208900_")
	if state.ReadUint32(key) == 0 {

		for _, participant := range participantsFixDoubleDelegationReward8208900 {
			addressBytes, err := hex.DecodeString(participant.address)
			if err != nil {
				panic(fmt.Errorf("cannot parse %s , err %s", participant.address, err))
			}
			wrongRewardValue := state.ReadUint64(_formatCumulativeParticipationReward(addressBytes))
			state.WriteUint64(_formatCumulativeParticipationReward(addressBytes), uint64(participant.reward))
			correctRewardValue := state.ReadUint64(_formatCumulativeParticipationReward(addressBytes))
			fmt.Printf("elections fix rewards: Participant %s reward changed from %d to %d\n", participant.address, wrongRewardValue, correctRewardValue)
		}

		for _, guardian := range guardiansFixDoubleDelegationReward8208900 {
			addressBytes, err := hex.DecodeString(guardian.address)
			if err != nil {
				panic(fmt.Errorf("cannot parse %s , err %s", guardian.address, err))
			}
			wrongRewardValue := state.ReadUint64(_formatCumulativeGuardianExcellenceReward(addressBytes))
			state.WriteUint64(_formatCumulativeGuardianExcellenceReward(addressBytes), uint64(guardian.reward))
			correctRewardValue := state.ReadUint64(_formatCumulativeGuardianExcellenceReward(addressBytes))
			fmt.Printf("elections fix rewards: Guardian %s reward changed from %d to %d\n", guardian.address, wrongRewardValue, correctRewardValue)
		}

		state.WriteUint32(key, 1)
	} else {
		panic(fmt.Sprintf("cannot fix rewards anymore"))
	}
}

var doubleDelegators8208900 = []string{
	"f7aba9b064a12330a00eafaa930e2fe8e76e65f0",
}

func _fixDelegatorState8208900() {
	key := []byte("_fixDelegatorState_8208900_")
	if state.ReadUint32(key) == 0 {

		doubleDelegatorMap := make(map[[20]byte]bool, len(doubleDelegators8208900))
		for _, delegator := range doubleDelegators8208900 {
			addressBytes, err := hex.DecodeString(delegator)
			if err != nil {
				panic(fmt.Errorf("cannot parse %s, err %s", delegator, err))
			}
			doubleDelegatorMap[_addressSliceToArray(addressBytes)] = true
		}

		numOfDelegators := _getNumberOfDelegators()
		newNumofDelegators := numOfDelegators

		for i := 0; i < newNumofDelegators; i++ {
			delegator := _getDelegatorAtIndex(i)
			if doubleDelegatorMap[delegator] {
				_setDelegatorAtIndex(i, state.ReadBytes(_formatDelegatorIterator(newNumofDelegators-1)))
				fmt.Printf("elections fix delegators: Delegator pointer at %d is duplicate moving the pointer from %d instead\n", i, newNumofDelegators-1)
				delete(doubleDelegatorMap, delegator)
				newNumofDelegators--
			}
		}

		fmt.Printf("elections fix delegators: Removing un-needed pointers from %d to %d\n", newNumofDelegators, numOfDelegators)
		for i := newNumofDelegators; i < numOfDelegators; i++ {
			state.Clear(_formatDelegatorIterator(newNumofDelegators))
		}
		_setNumberOfDelegators(newNumofDelegators)
		fmt.Printf("elections fix delegators: New number of Delegations %d\n", _getNumberOfDelegators())

		state.WriteUint32(key, 1)
	} else {
		panic(fmt.Sprintf("cannot fix delegate state for double delegations anymore"))
	}
}
