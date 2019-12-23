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

These two function fix the state for rewards and delegation and can be called only once (and between election 9108900 and 9128900).

Fix reward drift in participation and guardians due to a bug in reading delegation event after 1.3.2 update (including update of geth)

_fixRewardsDrift9108900 : fixes the 20 delegators and 9 guardians rewards according to addresses

*/

var participantsFixMissingDelegationReward9108900 = []struct {
	address string
	reward  int
}{
	{"63AEf7616882F488BCa97361d1c24F05B4657ae5", 2158483}, // instead of 2158485
	{"8e2ee3400af08b6767cb7db17d58471816f9c157", 193125}, // instead of 190536
	{"55544cdef4b63e61a8735c64d0c7dde0fa9f82f5", 2538}, // instead of 0
	{"027a46c5d6a25c975620ee56f0cc42fe0a45354f", 1664}, // instead of 0
	{"6302dd9d0906ebd43230e35149da772386c7ead7", 1902}, // instead of 0
	{"978357ddf6ca1aecb02906590729f0afe86f7af5", 508}, // instead of 0
	{"aecd4991bd2ace684e7966d358799f4944f5512b", 474}, // instead of 0
	{"637a21d3a830b2181741e9bb7e8c947ea0a4b6a4", 420}, // instead of 0
	{"fc797d9d38d7cf34f92247104d9b546d80f92f83", 420}, // instead of 0
	{"001576a667eb3d8331034303fe68f4e44b168d5b", 350}, // instead of 0
	{"848ddc306cca2f89630fb759bf6ee0b77784cb84", 62}, // instead of 0
	{"2fa676f4d6bdbba48a52069c58a986b9b9efe3fe", 69375}, // instead of 69378
	{"aff328c702e24a16830f30182009afb3126c7561", 38113}, // instead of 38115
	{"5902083148fed7036d7ae22b291e2d74655bd7e8", 2}, // instead of 0
}

var guardiansFixMissingDelegationReward9108900 = []struct {
	address string
	reward  int
}{
	{"63AEf7616882F488BCa97361d1c24F05B4657ae5", 5317670}, // instead of 5299749
	{"f7ae622C77D0580f02Bcb2f92380d61e3F6e466c", 5665367}, // instead of 5676231
	{"C5e624d6824e626a6f14457810E794E4603CFee2", 1308570}, // instead of 1310575
	{"8287928a809346dF4Cd53A096025a1136F7C4fF5", 3671176}, // instead of 3672718
	{"F058cCFB2324310C33E8FD9a1ddA8E99C8bEdA59", 2302600}, // instead of 2303541
	{"9afc8EF233e2793B2b90Ca5d70cA2e7098013142", 1849116}, // instead of 1849934
	{"C82eC0F3c834d337c51c9375a1C0A65Ce7aaDaec", 2548641}, // instead of 2549294
	{"ca2229dD9B45CA69E85Fd7119b2dF292BC4B8EFB", 1049747}, // instead of 1050208
	{"0874BC1383958e2475dF73dC68C4F09658E23777", 905542}, // instead of 905909
	{"f257EDE1CE68CA4b94e18eae5CB14942CBfF7D1C", 256395}, // instead of 256485
}

func _fixRewardsDrift9108900() {
	key := []byte("_fixRewards9108900_")
	if state.ReadUint32(key) == 0 {

		for _, participant := range participantsFixMissingDelegationReward9108900 {
			addressBytes, err := hex.DecodeString(participant.address)
			if err != nil {
				panic(fmt.Errorf("cannot parse %s , err %s", participant.address, err))
			}
			wrongRewardValue := state.ReadUint64(_formatCumulativeParticipationReward(addressBytes))
			state.WriteUint64(_formatCumulativeParticipationReward(addressBytes), uint64(participant.reward))
			correctRewardValue := state.ReadUint64(_formatCumulativeParticipationReward(addressBytes))
			fmt.Printf("elections fix rewards: Participant %s reward changed from %d to %d\n", participant.address, wrongRewardValue, correctRewardValue)
		}

		for _, guardian := range guardiansFixMissingDelegationReward9108900 {
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
