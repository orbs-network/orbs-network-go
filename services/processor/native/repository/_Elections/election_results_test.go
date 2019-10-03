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

func TestOrbsElectionResultsContract_isElectionOverDue_Yes(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		electionTime := startTimeBasedGetElectionTime()
		m.MockEthereumGetBlockTime(int(electionTime) + 3*int(MIRROR_PERIOD_LENGTH_IN_NANOS) + 1)

		// call
		b := isElectionOverdue()

		// assert
		require.EqualValues(t, 1, b)
	})
}

func TestOrbsElectionResultsContract_isElectionOverDue_No(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		electionTime := startTimeBasedGetElectionTime()
		m.MockEthereumGetBlockTime(int(electionTime) + 3*int(MIRROR_PERIOD_LENGTH_IN_NANOS) - 1)

		// call
		b := isElectionOverdue()

		// assert
		require.EqualValues(t, 0, b)
	})
}

func TestOrbsElectionResultsContract_updateElectionResults(t *testing.T) {
	currIndex := uint32(2)
	currTime := uint64(50000)
	currBlockNumber := uint64(10000)
	currentBlockHeight := uint64(1000)
	currElected := []byte{0x01}
	currOrbsElected := []byte{0xa1}
	newTime := uint64(60000)
	newBlockNumber := uint64(20000)
	newElected := [][20]byte{{0x02}}
	newElectedOrbs := [][20]byte{{0xb2}}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setPastElection(currIndex, currTime, currBlockNumber, currentBlockHeight, currElected, currOrbsElected)
		_setNumberOfElections(currIndex)
		_setValidatorOrbsAddress(newElected[0][:], newElectedOrbs[0][:])
		m.MockEnvBlockHeight(5000000)

		// call
		_setElectedValidators(newElected, newTime, newBlockNumber)

		// assert
		require.EqualValues(t, currIndex+1, getNumberOfElections())
		require.EqualValues(t, newTime, getElectedValidatorsTimeInNanosByIndex(currIndex+1))
		require.EqualValues(t, newBlockNumber, getElectedValidatorsBlockNumberByIndex(currIndex+1))
		require.EqualValues(t, _translateElectedAddressesToOrbsAddressesAndConcat(newElected), getElectedValidatorsOrbsAddressByIndex(currIndex+1))
		require.EqualValues(t, _concatElectedEthereumAddresses(newElected), getElectedValidatorsEthereumAddressByIndex(currIndex+1))
		require.EqualValues(t, currTime, getElectedValidatorsTimeInNanosByIndex(currIndex))
		require.EqualValues(t, currBlockNumber, getElectedValidatorsBlockNumberByIndex(currIndex))
		require.EqualValues(t, currentBlockHeight, getElectedValidatorsBlockHeightByIndex(currIndex))
		require.EqualValues(t, currElected, getElectedValidatorsEthereumAddressByIndex(currIndex))
		require.EqualValues(t, currOrbsElected, getElectedValidatorsOrbsAddressByIndex(currIndex))
	})
}

func TestOrbsElectionResultsContract_updateElectionResults_Empty(t *testing.T) {
	newTime := uint64(30000)
	newBlockNumber := uint64(20000)
	newElected := [][20]byte{{0x02}}
	newElectedOrbs := [][20]byte{{0xb2}}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setValidatorOrbsAddress(newElected[0][:], newElectedOrbs[0][:])
		m.MockEnvBlockHeight(5000000)

		// call
		_setElectedValidators(newElected, newTime, newBlockNumber)

		// assert
		require.EqualValues(t, 1, getNumberOfElections())
		require.EqualValues(t, newTime, getElectedValidatorsTimeInNanosByIndex(1))
		require.EqualValues(t, newBlockNumber, getElectedValidatorsBlockNumberByIndex(1))
		require.EqualValues(t, _translateElectedAddressesToOrbsAddressesAndConcat(newElected), getElectedValidatorsOrbsAddressByIndex(1))
		require.EqualValues(t, _concatElectedEthereumAddresses(newElected), getElectedValidatorsEthereumAddressByIndex(1))
	})
}

func TestOrbsElectionResultsContract_updateElectionResults_WrongBlockNumber(t *testing.T) {
	currIndex := uint32(2)
	currBlockNumber := uint64(10000)
	newTime := uint64(500)
	newBlockNumber := uint64(500)
	newElected := [][20]byte{{0x02}}

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		_setElectedValidatorsBlockNumberAtIndex(currIndex, currBlockNumber)
		_setNumberOfElections(currIndex)
		m.MockEnvBlockHeight(5000000)

		// call
		require.Panics(t, func() {
			_setElectedValidators(newElected, newTime, newBlockNumber)
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
		setPastElection(1, 100, blockNumber1, 1, elected1, electedOrbs1)
		setPastElection(2, 200, blockNumber2, 100, elected2, electedOrbs2)
		setPastElection(3, 300, blockNumber3, 200, elected3, electedOrbs3)

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
		setPastElection(1, 100, 100, blockHeight1, elected1, electedOrbs1)
		setPastElection(2, 200, 200, blockHeight2, elected2, electedOrbs2)
		setPastElection(3, 300, 300, blockHeight3, elected3, electedOrbs3)

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

func setPastElection(index uint32, time uint64, blockNumber uint64, blockHeight uint64, elected []byte, electedOrbs []byte) {
	_setElectedValidatorsTimeInNanosAtIndex(index, time)
	_setElectedValidatorsBlockNumberAtIndex(index, blockNumber)
	_setElectedValidatorsBlockHeightAtIndex(index, blockHeight)
	_setElectedValidatorsOrbsAddressAtIndex(index, electedOrbs)
	_setElectedValidatorsEthereumAddressAtIndex(index, elected)
}
