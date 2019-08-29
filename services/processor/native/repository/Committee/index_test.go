// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	elections_systemcontract "github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
	"github.com/stretchr/testify/require"
	"testing"
)

func makeNodeAddress(a int) []byte {
	addr := make([]byte, digest.NODE_ADDRESS_SIZE_BYTES)
	addr[0] = byte(a)
	return addr
}

func makeNodeAddressArray(n int) [][]byte {
	addrs := make([][]byte, 0, n)
	for i := 1; i <= n;i++ {
		addrs = append(addrs, makeNodeAddress(i))
	}
	return addrs
}

func TestOrbsCommitteeContract_getReputaion(t *testing.T) {
	addr := []byte{0xa1}
	addr2 := []byte{0xa2}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// Prepare
		state.WriteUint32(_formatReputation(addr), 5)

		// assert
		addrRep := _getReputation(addr)
		require.True(t, addrRep == 5, "read value for address was not correct")
		addr2Rep := _getReputation(addr2)
		require.True(t, addr2Rep == 0, "new addr should start with 0 reputation")
	})
}

func TestOrbsCommitteeContract_degradeReputaion(t *testing.T) {
	addr := []byte{0xa1}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// Prepare
		moreThanRepCap := int(ReputationBottomCap) + 2

		// assert
		currRep := _getReputation(addr)
		require.True(t, currRep == 0, "new addr should start with 0 reputation")
		for i := 0; i < moreThanRepCap;i++ {
			_degradeReputation(addr)
			newRep := _getReputation(addr)
			if i < int(ReputationBottomCap) {
				require.EqualValues(t, currRep + 1, newRep, "call to degrade should add 1 to reputation")
			} else {
				require.EqualValues(t, ReputationBottomCap, newRep, "cannot go over cap of reputation")
			}
			currRep = newRep
		}
	})
}

func TestOrbsCommitteeContract_getElectedValidators(t *testing.T) {
	addrs := [][]byte{
		makeNodeAddress(100),
		makeNodeAddress(10),
		makeNodeAddress(254),
		makeNodeAddress(17),
		makeNodeAddress(66),
		makeNodeAddress(8),
		makeNodeAddress(18),
	}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		m.MockServiceCallMethod(elections_systemcontract.CONTRACT_NAME, elections_systemcontract.METHOD_GET_ELECTED_VALIDATORS, []interface{}{_concat(addrs)})

		// run with empty seed
		electedConcatenated := _getElectedValidators()

		//assert
		require.EqualValues(t, addrs, _split(electedConcatenated))
	})
}
