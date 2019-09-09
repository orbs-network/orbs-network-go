// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsCommitteeContract_updateMisses_HappyFlow(t *testing.T) {
	callerAddress := []byte{0x01}
	addrs := makeNodeAddressArray(1) // only one addr so that there is no ordering this test only checks the no-panic

	InServiceScope(nil, callerAddress, func(m Mockery) {
		_init()

		// prepare
		m.MockCallContractAddress(TRIGGER_CONTRACT, callerAddress)
		m.MockServiceCallMethod(elections_systemcontract.CONTRACT_NAME, elections_systemcontract.METHOD_GET_ELECTED_VALIDATORS, []interface{}{addrs[0]})
		m.MockEnvBlockProposerAddress(addrs[0])
		m.MockEnvBlockHeight(155)
		m.MockEmitEvent(CommitteeMemberReputationSetEvent, addrs[0], uint32(0))

		// run & assert
		require.NotPanics(t, func() {
			updateMisses()
		}, "should not panic because it found who to update in committee")
	})
}

func TestOrbsCommitteeContract_updateMisses_SignerExistsPanics(t *testing.T) {
	signerAddress := AnAddress()

	InServiceScope(signerAddress, nil, func(m Mockery) {
		_init()

		// run & assert
		require.Panics(t, func() {
			updateMisses()
		}, "should panic because a signer exists")
	})
}

func TestOrbsCommitteeContract_updateMisses_CallerNotTriggerPanics(t *testing.T) {
	callerAddress := AnAddress()

	InServiceScope(nil, callerAddress, func(m Mockery) {
		_init()

		// prepare
		m.MockCallContractAddress(TRIGGER_CONTRACT, []byte{0x01})

		// run & assert
		require.Panics(t, func() {
			updateMisses()
		}, "should panic because a caller that is not tirgger exits exists")
	})
}

func TestOrbsCommitteeContract_updateMisses_BlockProducerNotFoundPanics(t *testing.T) {
	callerAddress := []byte{0x01}
	addrs := makeNodeAddressArray(1) // only one addr so that there is no ordering this test only checks the no-panic

	InServiceScope(nil, callerAddress, func(m Mockery) {
		_init()

		// prepare
		m.MockCallContractAddress(TRIGGER_CONTRACT, callerAddress)
		m.MockServiceCallMethod(elections_systemcontract.CONTRACT_NAME, elections_systemcontract.METHOD_GET_ELECTED_VALIDATORS, []interface{}{addrs[0]})
		m.MockEnvBlockProposerAddress(makeNodeAddress(77)) // non-committee address
		m.MockEnvBlockHeight(155)

		// run & aassert
		require.Panics(t, func() {
			updateMisses()
		}, "should panic because proposer is not part of committee")
	})
}

func TestOrbsCommitteeContract_updateOrderedCommittee(t *testing.T) {
	addrs := makeNodeAddressArray(8)
	blockProposerInd := 3

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare // all have
		for i, addr := range addrs {
			_addMiss(addr)
			if i < blockProposerInd {
				m.MockEmitEvent(CommitteeMemberReputationSetEvent, addr, uint32(2))
			} else if i == blockProposerInd {
				m.MockEmitEvent(CommitteeMemberReputationSetEvent, addr, uint32(0))
			}
		}

		// run
		_updateOrderedCommittee(addrs, addrs[blockProposerInd])

		//assert
		for i, addr := range addrs {
			reputaion := getMisses(addr)
			if i < blockProposerInd {
				require.EqualValues(t, 2, reputaion)
			} else if i == blockProposerInd {
				require.EqualValues(t, 0, reputaion)
			} else {
				require.EqualValues(t, 1, reputaion)
			}
		}
	})
}

func TestOrbsCommitteeContract_updateOrderedCommittee_notFound(t *testing.T) {
	addrs := makeNodeAddressArray(8)
	blockProposer := makeNodeAddress(25)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		for _, addr := range addrs {
			m.MockEmitEvent(CommitteeMemberReputationSetEvent, addr, uint32(1))
		}

		// run
		_updateOrderedCommittee(addrs, blockProposer)

		// assert done for emit by InService
	})
}

func TestOrbsCommitteeContract_isMemberOfOrderedCommittee(t *testing.T) {
	addrs := makeNodeAddressArray(8)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// run & assert
		for _, addr := range addrs {
			require.True(t, _isMemberOfOrderedCommittee(addrs, addr))
		}
	})
}

func TestOrbsCommitteeContract_isMemberOfOrderedCommittee_notFound(t *testing.T) {
	addrs := makeNodeAddressArray(1)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// run & assert
		for i := 2; i < 256;i++ {
			require.False(t, _isMemberOfOrderedCommittee(addrs, makeNodeAddress(i)))
		}
	})
}

