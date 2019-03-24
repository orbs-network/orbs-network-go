// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSecondsToNano(t *testing.T) {
	require.EqualValues(t, 33000000000, secondsToNano(33))
}

func TestAlgoDidReachResult(t *testing.T) {
	tests := []struct {
		name        string
		referenceTs primitives.TimestampNano
		below       BlockNumberAndTime
		above       BlockNumberAndTime
		expected    bool
	}{
		{
			name:        "Middle",
			referenceTs: 1022,
			below:       BlockNumberAndTime{BlockNumber: 33, BlockTimeNano: 1019},
			above:       BlockNumberAndTime{BlockNumber: 34, BlockTimeNano: 1025},
			expected:    true,
		},
		{
			name:        "Edge",
			referenceTs: 1022,
			below:       BlockNumberAndTime{BlockNumber: 33, BlockTimeNano: 1022},
			above:       BlockNumberAndTime{BlockNumber: 34, BlockTimeNano: 1025},
			expected:    true,
		},
		{
			name:        "MiddleButNotConsecutive",
			referenceTs: 1022,
			below:       BlockNumberAndTime{BlockNumber: 32, BlockTimeNano: 1019},
			above:       BlockNumberAndTime{BlockNumber: 34, BlockTimeNano: 1025},
			expected:    false,
		},
		{
			name:        "IncorrectEdge",
			referenceTs: 1022,
			below:       BlockNumberAndTime{BlockNumber: 33, BlockTimeNano: 1019},
			above:       BlockNumberAndTime{BlockNumber: 34, BlockTimeNano: 1022},
			expected:    false,
		},
		{
			name:        "AllEqual",
			referenceTs: 1022,
			below:       BlockNumberAndTime{BlockNumber: 33, BlockTimeNano: 1022},
			above:       BlockNumberAndTime{BlockNumber: 34, BlockTimeNano: 1022},
			expected:    false,
		},
		{
			name:        "MiddleButZero",
			referenceTs: 1022,
			below:       BlockNumberAndTime{BlockNumber: 0, BlockTimeNano: 0},
			above:       BlockNumberAndTime{BlockNumber: 1, BlockTimeNano: 1025},
			expected:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, algoDidReachResult(test.referenceTs, test.below, test.above))
		})
	}
}

func TestAlgoVerifyResultInsideRange(t *testing.T) {
	tests := []struct {
		name          string
		referenceTs   primitives.TimestampNano
		below         BlockNumberAndTime
		above         BlockNumberAndTime
		expectedError bool
	}{
		{
			name:          "Success",
			referenceTs:   1022,
			below:         BlockNumberAndTime{BlockNumber: 3, BlockTimeNano: 1000},
			above:         BlockNumberAndTime{BlockNumber: 400, BlockTimeNano: 1099},
			expectedError: false,
		},
		{
			name:          "TheResult",
			referenceTs:   1022,
			below:         BlockNumberAndTime{BlockNumber: 33, BlockTimeNano: 1019},
			above:         BlockNumberAndTime{BlockNumber: 34, BlockTimeNano: 1025},
			expectedError: true,
		},
		{
			name:          "UpsideDown",
			referenceTs:   1022,
			below:         BlockNumberAndTime{BlockNumber: 35, BlockTimeNano: 1025},
			above:         BlockNumberAndTime{BlockNumber: 34, BlockTimeNano: 1019},
			expectedError: true,
		},
		{
			name:          "ZeroBelow",
			referenceTs:   1022,
			below:         BlockNumberAndTime{BlockNumber: 0, BlockTimeNano: 0},
			above:         BlockNumberAndTime{BlockNumber: 34, BlockTimeNano: 1025},
			expectedError: true,
		},
		{
			name:          "OutsideRangeBelow",
			referenceTs:   77,
			below:         BlockNumberAndTime{BlockNumber: 3, BlockTimeNano: 1000},
			above:         BlockNumberAndTime{BlockNumber: 400, BlockTimeNano: 1099},
			expectedError: true,
		},
		{
			name:          "OutsideRangeAbove",
			referenceTs:   2099,
			below:         BlockNumberAndTime{BlockNumber: 3, BlockTimeNano: 1000},
			above:         BlockNumberAndTime{BlockNumber: 400, BlockTimeNano: 1099},
			expectedError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := algoVerifyResultInsideRange(test.referenceTs, test.below, test.above)
			if !test.expectedError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAlgoExtendAbove(t *testing.T) {
	tests := []struct {
		name          string
		referenceTs   primitives.TimestampNano
		btg           BlockTimeGetter
		expectedError bool
		expectedAbove int64
	}{
		{
			name:          "Success",
			referenceTs:   1022,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1033}, {2, 1044}, {3, 1055}}),
			expectedError: false,
			expectedAbove: 3,
		},
		{
			name:          "NoBlocks",
			referenceTs:   1022,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{}),
			expectedError: true,
			expectedAbove: 0,
		},
		{
			name:          "NoEnoughNewBlocksButOnEdge",
			referenceTs:   1055,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1033}, {2, 1044}, {3, 1055}}),
			expectedError: true,
			expectedAbove: 0,
		},
		{
			name:          "NoEnoughNewBlocksFarFromEdge",
			referenceTs:   1066,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1033}, {2, 1044}, {3, 1055}}),
			expectedError: true,
			expectedAbove: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			above, err := algoExtendAbove(context.TODO(), test.referenceTs, test.btg)
			if !test.expectedError {
				require.NoError(t, err)
				require.EqualValues(t, test.expectedAbove, above.BlockNumber)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAlgoExtendBelow(t *testing.T) {
	tests := []struct {
		name          string
		referenceTs   primitives.TimestampNano
		belowBlockNum int64
		aboveBlockNum int64
		btg           BlockTimeGetter
		expectedError bool
		expectedBelow int64
	}{
		{
			name:          "SuccessBlocksBelow1000",
			referenceTs:   1066,
			belowBlockNum: 33,
			aboveBlockNum: 34,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1033}, {2, 1044}, {3, 1055}}),
			expectedError: false,
			expectedBelow: 1,
		},
		{
			name:          "SuccessBlocks1000To10000",
			referenceTs:   1066,
			belowBlockNum: 2 + TIMESTAMP_FINDER_PROBABLE_RANGE_EFFICIENT,
			aboveBlockNum: 3 + TIMESTAMP_FINDER_PROBABLE_RANGE_EFFICIENT,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1033}, {2, 1044}, {3, 1055}}),
			expectedError: false,
			expectedBelow: 2,
		},
		{
			name:          "SuccessBlocksAbove10000",
			referenceTs:   1066,
			belowBlockNum: 2 + TIMESTAMP_FINDER_PROBABLE_RANGE_INEFFICIENT,
			aboveBlockNum: 3 + TIMESTAMP_FINDER_PROBABLE_RANGE_INEFFICIENT,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1033}, {2, 1044}, {3, 1055}, {2 + TIMESTAMP_FINDER_PROBABLE_RANGE_INEFFICIENT - TIMESTAMP_FINDER_PROBABLE_RANGE_EFFICIENT, 9999}}),
			expectedError: false,
			expectedBelow: 2,
		},
		{
			name:          "SuccessBlocksAbove10000_NoStartingPoint",
			referenceTs:   1066,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1033}, {2, 1044}, {3, 1055}, {2 + TIMESTAMP_FINDER_PROBABLE_RANGE_INEFFICIENT - TIMESTAMP_FINDER_PROBABLE_RANGE_EFFICIENT, 9999}}),
			expectedError: false,
			expectedBelow: 1,
		},
		{
			name:          "NoBlocks",
			referenceTs:   1022,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{}),
			expectedError: true,
			expectedBelow: 0,
		},
		{
			name:          "FirstBlockIsNewer",
			referenceTs:   1022,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1066}}),
			expectedError: true,
			expectedBelow: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			below, err := algoExtendBelow(context.TODO(), test.referenceTs, test.belowBlockNum, test.aboveBlockNum, test.btg)
			if !test.expectedError {
				require.NoError(t, err)
				require.EqualValues(t, test.expectedBelow, below.BlockNumber)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAlgoReduceRange(t *testing.T) {
	tests := []struct {
		name          string
		referenceTs   primitives.TimestampNano
		below         BlockNumberAndTime
		above         BlockNumberAndTime
		btg           BlockTimeGetter
		step          int
		expectedError bool
		expectedBelow int64
		expectedAbove int64
	}{
		{
			name:          "RangeOf3_BinarySearch_Upper",
			referenceTs:   1750,
			below:         BlockNumberAndTime{33, 1000},
			above:         BlockNumberAndTime{35, 2000},
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{34, 1500}}),
			step:          100, // binary search
			expectedError: false,
			expectedBelow: 34,
			expectedAbove: 35,
		},
		{
			name:          "RangeOf3_BinarySearch_Lower",
			referenceTs:   1250,
			below:         BlockNumberAndTime{33, 1000},
			above:         BlockNumberAndTime{35, 2000},
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{34, 1500}}),
			step:          100, // binary search
			expectedError: false,
			expectedBelow: 33,
			expectedAbove: 34,
		},
		{
			name:          "RangeOf3_Heuristic_Upper",
			referenceTs:   1750,
			below:         BlockNumberAndTime{33, 1000},
			above:         BlockNumberAndTime{35, 2000},
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{34, 1500}}),
			step:          1, // heuristic
			expectedError: false,
			expectedBelow: 34,
			expectedAbove: 35,
		},
		{
			name:          "RangeOf3_Heuristic_Lower",
			referenceTs:   1250,
			below:         BlockNumberAndTime{33, 1000},
			above:         BlockNumberAndTime{35, 2000},
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{34, 1500}}),
			step:          1, // heuristic
			expectedError: false,
			expectedBelow: 33,
			expectedAbove: 34,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			below, above, err := algoReduceRange(context.TODO(), test.referenceTs, test.below, test.above, test.btg, test.step)
			if !test.expectedError {
				require.NoError(t, err)
				require.EqualValues(t, test.expectedBelow, below.BlockNumber)
				require.EqualValues(t, test.expectedAbove, above.BlockNumber)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAlgoGetCursorWithBinarySearch(t *testing.T) {
	tests := []struct {
		name     string
		below    BlockNumberAndTime
		above    BlockNumberAndTime
		expected int64
	}{
		{
			name:     "5_7",
			below:    BlockNumberAndTime{BlockNumber: 5, BlockTimeNano: 1019},
			above:    BlockNumberAndTime{BlockNumber: 7, BlockTimeNano: 1025},
			expected: 6,
		},
		{
			name:     "4_6",
			below:    BlockNumberAndTime{BlockNumber: 4, BlockTimeNano: 1019},
			above:    BlockNumberAndTime{BlockNumber: 6, BlockTimeNano: 1025},
			expected: 5,
		},
		{
			name:     "5_8",
			below:    BlockNumberAndTime{BlockNumber: 5, BlockTimeNano: 1019},
			above:    BlockNumberAndTime{BlockNumber: 8, BlockTimeNano: 1025},
			expected: 6,
		},
		{
			name:     "4_7",
			below:    BlockNumberAndTime{BlockNumber: 4, BlockTimeNano: 1019},
			above:    BlockNumberAndTime{BlockNumber: 7, BlockTimeNano: 1025},
			expected: 5,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, algoGetCursorWithBinarySearch(test.below, test.above))
		})
	}
}

func TestAlgoGetCursorWithHeuristics(t *testing.T) {
	tests := []struct {
		name        string
		referenceTs primitives.TimestampNano
		below       BlockNumberAndTime
		above       BlockNumberAndTime
		expected    int64
	}{
		{
			name:        "BlockDiff2000_Middle",
			referenceTs: 1500000000,
			below:       BlockNumberAndTime{BlockNumber: 5000, BlockTimeNano: 1000000000},
			above:       BlockNumberAndTime{BlockNumber: 7000, BlockTimeNano: 2000000000},
			expected:    6000,
		},
		{
			name:        "BlockDiff2000_75Percent",
			referenceTs: 1750000000,
			below:       BlockNumberAndTime{BlockNumber: 5000, BlockTimeNano: 1000000000},
			above:       BlockNumberAndTime{BlockNumber: 7000, BlockTimeNano: 2000000000},
			expected:    6500,
		},
		{
			name:        "BlockDiff2000_25Percent",
			referenceTs: 1250000000,
			below:       BlockNumberAndTime{BlockNumber: 5000, BlockTimeNano: 1000000000},
			above:       BlockNumberAndTime{BlockNumber: 7000, BlockTimeNano: 2000000000},
			expected:    5500,
		},
		{
			name:        "BlockDiff2_Middle",
			referenceTs: 1500,
			below:       BlockNumberAndTime{BlockNumber: 5, BlockTimeNano: 1000},
			above:       BlockNumberAndTime{BlockNumber: 7, BlockTimeNano: 2000},
			expected:    6,
		},
		{
			name:        "BlockDiff2_LeftEdge",
			referenceTs: 1000,
			below:       BlockNumberAndTime{BlockNumber: 5, BlockTimeNano: 1000},
			above:       BlockNumberAndTime{BlockNumber: 7, BlockTimeNano: 2000},
			expected:    6,
		},
		{
			name:        "BlockDiff2_Left",
			referenceTs: 1001,
			below:       BlockNumberAndTime{BlockNumber: 5, BlockTimeNano: 1000},
			above:       BlockNumberAndTime{BlockNumber: 7, BlockTimeNano: 2000},
			expected:    6,
		},
		{
			name:        "BlockDiff2_RightEdge",
			referenceTs: 2000,
			below:       BlockNumberAndTime{BlockNumber: 5, BlockTimeNano: 1000},
			above:       BlockNumberAndTime{BlockNumber: 7, BlockTimeNano: 2000},
			expected:    6,
		},
		{
			name:        "BlockDiff2_Right",
			referenceTs: 1999,
			below:       BlockNumberAndTime{BlockNumber: 5, BlockTimeNano: 1000},
			above:       BlockNumberAndTime{BlockNumber: 7, BlockTimeNano: 2000},
			expected:    6,
		},
		{
			name:        "ZeroTimeDiff",
			referenceTs: 1000,
			below:       BlockNumberAndTime{BlockNumber: 5, BlockTimeNano: 1000},
			above:       BlockNumberAndTime{BlockNumber: 7, BlockTimeNano: 1000},
			expected:    6,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, algoGetCursorWithHeuristics(test.referenceTs, test.below, test.above))
		})
	}
}
