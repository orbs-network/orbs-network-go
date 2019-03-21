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
