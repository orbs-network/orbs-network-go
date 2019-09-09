// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsCommitteeContract_getMiss(t *testing.T) {
	addr := []byte{0xa1}
	addr2 := []byte{0xa2}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// Prepare
		state.WriteUint32(_formatMisses(addr), 5)

		// assert
		addrMiss := getMisses(addr)
		require.True(t, addrMiss == 5, "read value for address was not correct")
		addr2Miss := getMisses(addr2)
		require.True(t, addr2Miss == 0, "new addr should start with 0 reputation")
	})
}

func TestOrbsCommitteeContract_addMiss(t *testing.T) {
	addr := []byte{0xa1}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// Prepare
		moreThanRepCap := int(ReputationBottomCap) + 2

		// assert
		curr := getMisses(addr)
		require.True(t, curr == 0, "new addr should start with 0 misses")
		for i := 0; i < moreThanRepCap;i++ {
			_addMiss(addr)
			newMiss := getMisses(addr)
			require.EqualValues(t, curr+ 1, newMiss, "call to degrade should add 1 to misses")
			curr = newMiss
		}
	})
}

func TestOrbsCommitteeContract_getReputation(t *testing.T) {
	addr := []byte{0xa1}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// Prepare
		moreThanRepCap := int(ReputationBottomCap) + 2

		// assert
		for i := 0; i < moreThanRepCap;i++ {
			rep := getReputation(addr)
			miss := getMisses(addr)
			if i < int(ToleranceLevel) {
				require.EqualValues(t, 0, rep, "upto tolerance should be 0")
			} else if i < int(ReputationBottomCap) {
				require.EqualValues(t, miss, rep, "upto cap should be miss")
			} else {
				require.EqualValues(t, ReputationBottomCap, rep, "cannot go over cap of reputation")
			}
			_addMiss(addr)
		}
	})
}
