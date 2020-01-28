// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"bytes"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsCommitteeContract_check_random(t *testing.T) {
	totalWeight := 1280
	random := []byte{0x02, 0x1, 0x2, 0x3, 0x5, 0x5, 0x6, 0x4}
	maxRuns := 1000000
	res := make([]int, totalWeight+1)
	for i := 0; i < maxRuns; i++ {
		random = _nextRandom(random)
		ind := _getRandomWeight(random, totalWeight)
		res[ind] += 1
	}
	expected := maxRuns / totalWeight
	expectedOver := expected * 98 / 100
	expectedUnder := expected * 102 / 100
	tooFar := 0
	for i := 1; i <= totalWeight; i++ {
		if res[i] < expectedOver || res[i] > expectedUnder {
			tooFar++
		}
	}
	require.LessOrEqual(t, float64(tooFar)/float64(maxRuns), 0.001, "misses of+-2 percent should be under 0.1 percent")
}

func TestOrbsCommitteeContract_check_OrderWithDiffSeedAndRep(t *testing.T) {
	committeeSize := 5
	addressArray := makeNodeAddressArray(committeeSize)

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		ordered := _orderList(addressArray, []byte{7})
		orderedDiffSeed := _orderList(addressArray, []byte{8})
		require.NotEqual(t, ordered, orderedDiffSeed)
		require.ElementsMatch(t, ordered, orderedDiffSeed, "must actually be same list in diff order")
		orderedSameSeed := _orderList(addressArray, []byte{7})
		require.EqualValues(t, ordered, orderedSameSeed)
		state.WriteUint32(_formatMisses(addressArray[0]), ReputationBottomCap)
		orderedSameSeedDiffProb := _orderList(addressArray, []byte{7})
		require.NotEqual(t, ordered, orderedSameSeedDiffProb)
		require.ElementsMatch(t, ordered, orderedSameSeedDiffProb, "must actually be same list in diff order")
	})
}

func TestOrbsCommitteeContract_orderList_AllRepIs0(t *testing.T) {
	max := 10000
	committeeSize := 20
	addresses := makeNodeAddressArray(committeeSize)
	checkAddress := addresses[0]
	var foundAtPos0, foundAtPosMid, foundAtPosLast int

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		// no setup

		// run
		midLocation := committeeSize / 2
		endLocation := committeeSize - 1
		for i := 1 ; i <= max ; i++ {
			m.MockEnvBlockHeight(i) // to generate random seed
			outputArr := _orderList(addresses, _generateSeed())
			if bytes.Equal(outputArr[0], checkAddress) {
				foundAtPos0++
			} else if bytes.Equal(outputArr[midLocation], checkAddress){
				foundAtPosMid++
			} else if bytes.Equal(outputArr[endLocation], checkAddress){
				foundAtPosLast++
			}
		}

		// assert
		expected := max / committeeSize // equal chance
		requireCountToBeInRange(t, foundAtPos0, expected)
		requireCountToBeInRange(t, foundAtPosMid, expected)
		requireCountToBeInRange(t, foundAtPosLast, expected)
	})
}

func TestOrbsCommitteeContract_orderList_OneRepIsWorst(t *testing.T) {
	max := 100000
	committeeSize := 4 // to allow a good spread in a small amount of runs we need a small committee size.
	addresses := makeNodeAddressArray(committeeSize)
	badAddress := addresses[0]
	goodAddress := addresses[1]
	foundBadAddressInFirstPosition := 0
	foundGoodAddressInFirstPosition := 0

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		state.WriteUint32(_formatMisses(badAddress), ReputationBottomCap)

		// run
		for i := 1 ; i <= max ; i++ {
			m.MockEnvBlockHeight(i) // to generate random seed
			outputArr := _orderList(addresses, _generateSeed())
			if bytes.Equal(outputArr[0], badAddress) {
				foundBadAddressInFirstPosition++
			}
			if bytes.Equal(outputArr[0], goodAddress) {
				foundGoodAddressInFirstPosition++
			}
		}

		// assert
		requireCountToBeInRange(t, foundGoodAddressInFirstPosition, (max * 64) / ((committeeSize-1) * 64 + 1))
		requireCountToBeInRange(t, foundBadAddressInFirstPosition, max / ((committeeSize-1) * 64 + 1))
	})
}

func TestOrbsCommitteeContract_orderList_QuarterAreALittleBad(t *testing.T) {
	max := 10000
	committeeSize := 20
	addresses := makeNodeAddressArray(committeeSize)
	badAddress := addresses[0]
	goodAddress := addresses[5]
	foundBadAddressInFirstPosition := 0
	foundGoodAddressInFirstPosition := 0

	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// prepare
		state.WriteUint32(_formatMisses(addresses[0]), ToleranceLevel+2)
		state.WriteUint32(_formatMisses(addresses[1]), ToleranceLevel+2)
		state.WriteUint32(_formatMisses(addresses[2]), ToleranceLevel+2)
		state.WriteUint32(_formatMisses(addresses[3]), ToleranceLevel+2)
		state.WriteUint32(_formatMisses(addresses[4]), ToleranceLevel+2)

		// run
		for i := 1 ; i <= max ; i++ {
			m.MockEnvBlockHeight(i) // to generate random seed
			outputArr := _orderList(addresses, _generateSeed())
			if bytes.Equal(outputArr[0], badAddress) {
				foundBadAddressInFirstPosition++
			}
			if bytes.Equal(outputArr[0], goodAddress) {
				foundGoodAddressInFirstPosition++
			}
		}

		// assert
		requireCountToBeInRange(t, foundGoodAddressInFirstPosition, (max * 64) / ((committeeSize-5) * 64 + 5 * 16))
		requireCountToBeInRange(t, foundBadAddressInFirstPosition, (max * 16) / ((committeeSize-5) * 64 + 5 * 16))
	})
}

func requireCountToBeInRange(t testing.TB, actual, expected int) {
	require.InDelta(t, expected, actual, 0.05 * float64(expected), "expect (%d) to be five precent delta to (%d)", actual, expected)
}
