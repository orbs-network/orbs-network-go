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

func TestOrbsElectionResultsContract_getEffectiveElectionBlockNumber_emptyElection(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		b := getEffectiveElectionBlockNumber()

		// assert
		require.EqualValues(t, 0, b)
	})
}

func TestOrbsElectionResultsContract_getEffectiveElectionBlockNumber(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setPastElection(1, 10000, 50, []byte{}, []byte{})
		_setNumberOfElections(1)

		// call
		b := getEffectiveElectionBlockNumber()

		// assert
		require.EqualValues(t, 10000, b)
	})
}

func TestOrbsElectionResultsContract_updateElectionResults(t *testing.T) {
	currIndex := uint32(2)
	currBlockNumber := uint64(10000)
	currentBlockHeight := uint64(1000)
	currElected := []byte{0x01}
	currOrbsElected := []byte{0xa1}
	newBlockNumber := uint64(20000)
	newElected := [][20]byte{{0x02}}
	newElectedOrbs := [][20]byte{{0xb2}}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setPastElection(currIndex, currBlockNumber, currentBlockHeight, currElected, currOrbsElected)
		_setNumberOfElections(currIndex)
		_setCurrentElectionBlockNumber(newBlockNumber)
		_setValidatorOrbsAddress(newElected[0][:], newElectedOrbs[0][:])

		// call
		_setElectedValidators(newElected)

		// assert
		require.EqualValues(t, currIndex+1, getNumberOfElections())
		require.EqualValues(t, newBlockNumber, getElectedValidatorsBlockNumberByIndex(currIndex+1))
		require.EqualValues(t, _translateElectedAddressesToOrbsAddressesAndConcat(newElected), getElectedValidatorsOrbsAddressByIndex(currIndex+1))
		require.EqualValues(t, _concatElectedEthereumAddresses(newElected), getElectedValidatorsEthereumAddressByIndex(currIndex+1))
		require.EqualValues(t, currBlockNumber, getElectedValidatorsBlockNumberByIndex(currIndex))
		require.EqualValues(t, currentBlockHeight, getElectedValidatorsBlockHeightByIndex(currIndex))
		require.EqualValues(t, currElected, getElectedValidatorsEthereumAddressByIndex(currIndex))
		require.EqualValues(t, currOrbsElected, getElectedValidatorsOrbsAddressByIndex(currIndex))
	})
}

func TestOrbsElectionResultsContract_updateElectionResults_Empty(t *testing.T) {
	newBlockNumber := uint64(20000)
	newElected := [][20]byte{{0x02}}
	newElectedOrbs := [][20]byte{{0xb2}}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(newBlockNumber)
		_setValidatorOrbsAddress(newElected[0][:], newElectedOrbs[0][:])

		// call
		_setElectedValidators(newElected)

		// assert
		require.EqualValues(t, 1, getNumberOfElections())
		require.EqualValues(t, newBlockNumber, getElectedValidatorsBlockNumberByIndex(1))
		require.EqualValues(t, _translateElectedAddressesToOrbsAddressesAndConcat(newElected), getElectedValidatorsOrbsAddressByIndex(1))
		require.EqualValues(t, _concatElectedEthereumAddresses(newElected), getElectedValidatorsEthereumAddressByIndex(1))
	})
}

func TestOrbsElectionResultsContract_updateElectionResults_WrongBlockNumber(t *testing.T) {
	currIndex := uint32(2)
	currBlockNumber := uint64(10000)
	newBlockNumber := uint64(500)
	newElected := [][20]byte{{0x02}}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setCurrentElectionBlockNumber(newBlockNumber)
		_setElectedValidatorsBlockNumberAtIndex(currIndex, currBlockNumber)
		_setNumberOfElections(currIndex)

		// call
		require.Panics(t, func() {
			_setElectedValidators(newElected)
		}, "should panic because newer blocknumber is in past")
	})
}

func TestOrbsElectionResultsContract_getElectionResultsByBlockNumber_getSeveralValues(t *testing.T) {
	blockNumber1 := uint64(10000)
	elected1 := []byte{0x01}
	electedOrbs1 := []byte{0xa1}
	blockNumber2 := uint64(20000)
	elected2 := []byte{0x02}
	electedOrbs2 := []byte{0xa2}
	blockNumber3 := uint64(30000)
	elected3 := []byte{0x03}
	electedOrbs3 := []byte{0xa3}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setPastElection(1, blockNumber1, 1, elected1, electedOrbs1)
		setPastElection(2, blockNumber2, 100, elected2, electedOrbs2)
		setPastElection(3, blockNumber3, 200, elected3, electedOrbs3)

		_setNumberOfElections(3)

		// call
		foundElected1 := getElectedValidatorsEthereumAddressByBlockNumber(blockNumber1 + 1)
		foundElected2 := getElectedValidatorsEthereumAddressByBlockNumber(blockNumber2 + 5000)
		foundElected3 := getElectedValidatorsEthereumAddressByBlockNumber(blockNumber3 + 1000000)
		foundElected0 := getElectedValidatorsEthereumAddressByBlockNumber(5)

		// assert
		require.EqualValues(t, elected1, foundElected1)
		require.EqualValues(t, elected2, foundElected2)
		require.EqualValues(t, elected3, foundElected3)
		require.EqualValues(t, _getDefaultElectionResults(), foundElected0)
	})
}

func TestOrbsElectionResultsContract_getElectionResultsByBlockHeight_getSeveralValues(t *testing.T) {
	blockHeight1 := uint64(10000)
	elected1 := []byte{0x01}
	electedOrbs1 := []byte{0xa1}
	blockHeight2 := uint64(20000)
	elected2 := []byte{0x02}
	electedOrbs2 := []byte{0xa2}
	blockHeight3 := uint64(30000)
	elected3 := []byte{0x03}
	electedOrbs3 := []byte{0xa3}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setPastElection(1, 100, blockHeight1, elected1, electedOrbs1)
		setPastElection(2, 200, blockHeight2, elected2, electedOrbs2)
		setPastElection(3, 300, blockHeight3, elected3, electedOrbs3)

		_setNumberOfElections(3)

		// call
		foundElected1 := getElectedValidatorsOrbsAddressByBlockHeight(blockHeight1 + 1)
		foundElected2 := getElectedValidatorsOrbsAddressByBlockHeight(blockHeight2 + 5000)
		foundElected3 := getElectedValidatorsOrbsAddressByBlockHeight(blockHeight3 + 1000000)
		foundElected0 := getElectedValidatorsOrbsAddressByBlockHeight(5)

		// assert
		require.EqualValues(t, electedOrbs1, foundElected1)
		require.EqualValues(t, electedOrbs2, foundElected2)
		require.EqualValues(t, electedOrbs3, foundElected3)
		require.EqualValues(t, _getDefaultElectionResults(), foundElected0)
	})
}

func setPastElection(index uint32, blockNumber uint64, blockHeight uint64, elected []byte, electedOrbs []byte) {
	_setElectedValidatorsBlockNumberAtIndex(index, blockNumber)
	_setElectedValidatorsBlockHeightAtIndex(index, blockHeight)
	_setElectedValidatorsOrbsAddressAtIndex(index, electedOrbs)
	_setElectedValidatorsEthereumAddressAtIndex(index, elected)
}

func TestOrbsVotingContract_initCurrentElectionBlockNumber(t *testing.T) {
	tests := []struct {
		name                     string
		expectCurrentBlockNumber uint64
		ethereumBlockNumber      uint64
	}{
		{"before is 0", FIRST_ELECTION_BLOCK, 0},
		{"before is a small number", FIRST_ELECTION_BLOCK, 5000000},
		{"before is after first but before second", FIRST_ELECTION_BLOCK + ELECTION_PERIOD_LENGTH_IN_BLOCKS, FIRST_ELECTION_BLOCK + 5000},
		{"before is after second", FIRST_ELECTION_BLOCK + 2*ELECTION_PERIOD_LENGTH_IN_BLOCKS, FIRST_ELECTION_BLOCK + ELECTION_PERIOD_LENGTH_IN_BLOCKS + 5000},
	}
	for i := range tests {
		cTest := tests[i]
		t.Run(cTest.name, func(t *testing.T) {
			InServiceScope(nil, nil, func(m Mockery) {
				_init()
				_setCurrentElectionBlockNumber(0)
				m.MockEthereumGetBlockNumber(int(cTest.ethereumBlockNumber))
				after := _getCurrentElectionBlockNumber()
				require.EqualValues(t, cTest.expectCurrentBlockNumber, after, "'%s' failed ", cTest.name)
			})
		})
	}
}
