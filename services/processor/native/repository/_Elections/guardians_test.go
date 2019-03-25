// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsVotingContract_guardians_isGuardian(t *testing.T) {
	guardians := [][20]byte{{0x01}, {0x02}, {0x03}}
	notGuardian := [20]byte{0xa1}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		_setGuardians(guardians)

		// assert
		require.EqualValues(t, len(guardians), _getNumberOfGuardians())
		for i := 0; i < _getNumberOfGuardians(); i++ {
			require.True(t, _isGuardian(guardians[i]))
		}
		require.False(t, _isGuardian(notGuardian))
	})
}

func TestOrbsVotingContract_guardians_setTwiceWithSmallerList(t *testing.T) {
	guardians := [][20]byte{{0x01}, {0x02}, {0x03}}
	guardians2 := [][20]byte{{0x01}, {0x03}}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		_setGuardians(guardians)
		_clearGuardians()
		_setGuardians(guardians2)

		// assert
		require.EqualValues(t, len(guardians2), _getNumberOfGuardians())
		for i := 0; i < _getNumberOfGuardians(); i++ {
			require.True(t, _isGuardian(guardians2[i]))
		}
		require.False(t, _isGuardian(guardians[1]))
	})
}

func TestOrbsVotingContract_guardians_setTwiceWithEmptyList(t *testing.T) {
	guardians := [][20]byte{{0x01}, {0x02}, {0x03}}
	guardians2 := make([][20]byte, 0)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		_setGuardians(guardians)
		_clearGuardians()
		_setGuardians(guardians2)

		// assert
		require.EqualValues(t, len(guardians2), _getNumberOfGuardians())
		for i := 0; i < len(guardians); i++ {
			require.False(t, _isGuardian(guardians[i]))
		}
	})
}

func TestOrbsVotingContract_processVote_clearGuardians(t *testing.T) {
	h := newHarness()
	h.electionBlock = uint64(60000)

	var g1 = h.addGuardian(100)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		h.setupOrbsStateBeforeProcess()
		_setCandidates(g1.address[:], [][20]byte{{0xdd}})
		_setGuardianStake(g1.address[:], 100)
		_setGuardianVoteBlockNumber(g1.address[:], h.electionBlock)
		_setCurrentElectionBlockNumber(h.electionBlock)

		// call
		_clearGuardians()
		guardians := _getGuardians()

		// assert
		m.VerifyMocks()
		require.EqualValues(t, 0, _getNumberOfGuardians())
		require.EqualValues(t, 0, getGuardianStake(g1.address[:]))
		require.EqualValues(t, 0, _getGuardianVoteBlockNumber(g1.address[:]))
		require.EqualValues(t, 0, len(guardians))
		require.EqualValues(t, [][20]byte{}, _getCandidates(g1.address[:]))
	})
}
