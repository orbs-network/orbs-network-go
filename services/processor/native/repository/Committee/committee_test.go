// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"encoding/binary"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/stretchr/testify/require"
	"testing"
)

func appendAndOrder(newAddr []byte, addrs [][]byte) [][]byte {
	return appendAndOrderWithSeed(newAddr, addrs, []byte{})
}

func appendAndOrderWithSeed(newAddr []byte, addrs [][]byte, seed []byte) [][]byte {
	currScore := _calculateScoreWithReputation(newAddr, seed)
	addrs = append(addrs, newAddr)
	for i := len(addrs) - 1; i > 0; i-- {
		if currScore > _calculateScoreWithReputation(addrs[i-1], seed) {
			addrs[i], addrs[i-1] = addrs[i-1], addrs[i]
		} else {
			break
		}
	}
	return addrs
}

func TestOrbsCommitteeContract_getOrderedCommittee_withoutReputation(t *testing.T) {
	addrs := makeNodeAddressArray(10)
	blockHeight := 155

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		m.MockEnvBlockHeight(blockHeight)

		// add each to the correct place
		expectedOrder := make([][]byte, 0, len(addrs))
		for _, addr := range addrs {
			expectedOrder = appendAndOrderWithSeed(addr, expectedOrder, _generateSeed())
		}

		// run
		ordered := _getOrderedCommitteeArray(addrs)

		//assert
		require.EqualValues(t, expectedOrder, ordered)
	})
}

func TestOrbsCommitteeContract_getOrderedCommittee_SimpleReputation(t *testing.T) {
	addrs := makeNodeAddressArray(3)
	blockHeight := 155

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		m.MockEnvBlockHeight(blockHeight)
		state.WriteUint32(_formatMisses(addrs[0]), 10)

		// sort with simplified calculation
		expectedOrder := make([][]byte, 0, len(addrs))
		for _, addr := range addrs {
			expectedOrder = appendAndOrderWithSeed(addr, expectedOrder, _generateSeed())
		}

		// run
		ordered := _getOrderedCommitteeArray(addrs)

		//assert
		require.EqualValues(t, expectedOrder, ordered)
	})
}

func TestOrbsCommitteeContract_orderList_noReputation_noSeed(t *testing.T) {
	addrs := makeNodeAddressArray(10)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// Prepare do calculation in similar way
		expectedOrder := make([][]byte, 0, len(addrs))
		for _, addr := range addrs {
			expectedOrder = appendAndOrder(addr, expectedOrder)
		}

		// run with empty seed
		ordered := _orderList(addrs, []byte{})

		//assert
		require.EqualValues(t, expectedOrder, ordered)
	})
}

func TestOrbsCommitteeContract_calculateScoreWithReputation(t *testing.T) {
	addr := makeNodeAddress(25)
	blockHeight := 777744444

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// Prepare
		m.MockEnvBlockHeight(blockHeight)
		scoreWithOutRep := _calculateScore(addr, _generateSeed())

		// rep below cap
		state.WriteUint32(_formatMisses(addr), 2)
		score := _calculateScoreWithReputation(addr, _generateSeed())
		require.EqualValues(t, scoreWithOutRep, score)

		// rep with factor (2^5)
		state.WriteUint32(_formatMisses(addr), 5)
		score = _calculateScoreWithReputation(addr, _generateSeed())
		require.EqualValues(t, float64(scoreWithOutRep)/float64(32), score)

		// rep with factor (2^10) miss is above cap
		state.WriteUint32(_formatMisses(addr), 11)
		score = _calculateScoreWithReputation(addr, _generateSeed())
		require.EqualValues(t, float64(scoreWithOutRep)/float64(1024), score)
	})
}

func TestOrbsCommitteeContract_calculateScore(t *testing.T) {
	addr := []byte{0xa1, 0x33}
	var emptySeed = []byte{}
	nonEmptySeed := []byte{0x44}
	nonEmptySeedOneBitDiff := []byte{0x43}

	scoreWithEmpty := _calculateScore(addr, emptySeed)
	scoreWithNonEmpty := _calculateScore(addr, nonEmptySeed)
	scoreWithNonEmptyOneBitDiff := _calculateScore(addr, nonEmptySeedOneBitDiff)

	shaOfAddrWithNoSeed := hash.CalcSha256(addr)
	expectedScoreWithEmpty := binary.LittleEndian.Uint32(shaOfAddrWithNoSeed[hash.SHA256_HASH_SIZE_BYTES-4:])

	require.Equal(t, expectedScoreWithEmpty, scoreWithEmpty, "for score with empty seed doesn't match expected")
	require.NotEqual(t, scoreWithNonEmpty, scoreWithEmpty, "for score with and without seed must not match")
	require.NotEqual(t, scoreWithNonEmpty, scoreWithNonEmptyOneBitDiff, "score is diff even with one bit difference in seed")
}
