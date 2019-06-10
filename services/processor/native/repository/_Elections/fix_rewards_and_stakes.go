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

_fixRewardsDrift7928900 : fixes the 20 delegators and 9 guardians rewards according to addresses
_fixDelegatorState7928900 : remove the double delegation pointer from the delegators' list in state.

*/

var participantsFixDoubleDelegationReward = []struct {
	address string
	reward  int
}{
	{"63cad78066b28c58f29f58a4f55827b2c2259776", 1350},  // instead of 2178
	{"aaa020747c3f27035a803a2a9b577e76ed5cdfc7", 441},   // instead of 693
	{"e4f60ade67862d6604cc7464621bc7ac17e75841", 220},   // instead of 352
	{"d5db86180abb0300334347a6a57bd318ffa7aebc", 860},   // instead of 1376
	{"9e912ba1ce059ceca9499ec9ec703b12d32cb50c", 860},   // instead of 1376
	{"606feb20c03a797a42390163dedd1d45297af73f", 680},   // instead of 973
	{"5d8c4be81a1203821b464bd76f43967f724d5004", 66},    // instead of 60
	{"a39fc2c9c437f2e56901ccc6bd18ee1100c82686", 902},   // instead of 1552
	{"683148b5e726a0c07561895e39f394c2d1ebf092", 99},    // instead of 90
	{"d7806c262ea5e3928d0970e2a0f19e931f9ed1af", 91751}, // instead of 83410
	{"fa76362c7937b02018abd99c04333d3b62ff7f3d", 624},   // instead of 1092
	{"9e2d2a783274c81337fec2e2897fee5ec614cca0", 539},   // instead of 490
	{"e933cedb1d7772ab3ca6ca8c173cc6b6ad61bb5f", 140},   // instead of 217
	{"0a91147313b82fdb20c853646e829b8d32a9fdb8", 1319},  // instead of 1879
	{"f07df0a74967c53611f2bb619eec462527ddb5d3", 297},   // instead of 270
	{"fddb0aa468c70be6440c46cfbdf099d2135e25a7", 2285},  // instead of 2287
	{"960ac2fd49b2088cfbb385fed12adb4d78429928", 6772},  // instead of 11800
	{"f061c6acc642676914aa7139abc89fbf97a06112", 20834}, // instead of 32198
	{"aa1a4565f2d875995d85e8d52e7193c666e20c04", 647},   // instead of 961
	{"8fd0a7b70aa896cf85b3034385f98af1d927d442", 4082},  // instead of 0
}

var guardiansFixDoubleDelegationReward = []struct {
	address string
	reward  int
}{
	{"f7ae622C77D0580f02Bcb2f92380d61e3F6e466c", 1399658}, // instead of 1403641
	{"63AEf7616882F488BCa97361d1c24F05B4657ae5", 1250192}, // instead of 1250217
	{"C82eC0F3c834d337c51c9375a1C0A65Ce7aaDaec", 865579},  // instead of 855152
	{"9afc8EF233e2793B2b90Ca5d70cA2e7098013142", 522112},  // instead of 522649
	{"8287928a809346dF4Cd53A096025a1136F7C4fF5", 340047},  // instead of 340157
	{"F058cCFB2324310C33E8FD9a1ddA8E99C8bEdA59", 527823},  // instead of 528491
	{"a3cBDD66267DAaA4B51Af6CD894c92054bb2F2c7", 21201},   // instead of 47243
	{"cB6172196BbCf5b4cf9949D7f2e4Ee802EF2b81D", 94199},   // instead of 85826
	{"1763F3DA9380E2df7FfDE1dC245801BB14F80669", 21887},   // instead of 15898
}

func _fixRewardsDrift7928900() {
	key := []byte("_fixRewards7928900_")
	if state.ReadUint32(key) == 0 {

		for _, participant := range participantsFixDoubleDelegationReward {
			bytes, err := hex.DecodeString(participant.address)
			if err != nil {
				panic(fmt.Errorf("cannot parse %s , err %s", participant.address, err))
			}
			wrongValue := state.ReadUint64(_formatCumulativeParticipationReward(bytes))
			state.WriteUint64(_formatCumulativeParticipationReward(bytes), uint64(participant.reward))
			newValue := state.ReadUint64(_formatCumulativeParticipationReward(bytes))
			fmt.Printf("elections fix rewards: Participant %s reward changed from %d to %d\n", participant.address, wrongValue, newValue)
		}

		for _, guardian := range guardiansFixDoubleDelegationReward {
			bytes, err := hex.DecodeString(guardian.address)
			if err != nil {
				panic(fmt.Errorf("cannot parse %s , err %s", guardian.address, err))
			}
			wrongValue := state.ReadUint64(_formatCumulativeGuardianExcellenceReward(bytes))
			state.WriteUint64(_formatCumulativeGuardianExcellenceReward(bytes), uint64(guardian.reward))
			newValue := state.ReadUint64(_formatCumulativeGuardianExcellenceReward(bytes))
			fmt.Printf("elections fix rewards: Guardian %s reward changed from %d to %d\n", guardian.address, wrongValue, newValue)
		}

		state.WriteUint32(key, 1)
	} else {
		panic(fmt.Sprintf("cannot fix rewards anymore"))
	}
}

var doubleDelegators = []string{
	"aaa020747c3f27035a803a2a9b577e76ed5cdfc7",
	"e933cedb1d7772ab3ca6ca8c173cc6b6ad61bb5f",
	"d5db86180abb0300334347a6a57bd318ffa7aebc",
	"9e912ba1ce059ceca9499ec9ec703b12d32cb50c",
	"e4f60ade67862d6604cc7464621bc7ac17e75841",
	"a39fc2c9c437f2e56901ccc6bd18ee1100c82686",
	"aa1a4565f2d875995d85e8d52e7193c666e20c04",
	"63cad78066b28c58f29f58a4f55827b2c2259776",
	"fa76362c7937b02018abd99c04333d3b62ff7f3d",
	"960ac2fd49b2088cfbb385fed12adb4d78429928",
	"f061c6acc642676914aa7139abc89fbf97a06112",
	"606feb20c03a797a42390163dedd1d45297af73f",
	"0a91147313b82fdb20c853646e829b8d32a9fdb8",
	"22afe11457c368ee1b0314477f0538bfb44843af",
}

func _fixDelegatorState7928900() {
	key := []byte("_fixDelegatorState_7928900_")
	if state.ReadUint32(key) == 0 {

		doubleDelegatorMap := make(map[[20]byte]bool, len(doubleDelegators))
		for _, delegator := range doubleDelegators {
			bytes, err := hex.DecodeString(delegator)
			if err != nil {
				panic(fmt.Errorf("cannot parse %s, err %s", delegator, err))
			}
			doubleDelegatorMap[_addressSliceToArray(bytes)] = true
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
