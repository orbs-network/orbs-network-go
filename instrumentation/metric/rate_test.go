// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func testRateMeasure(t *testing.T, measure func(rate *Rate)) {
	start := time.Now()
	rate := newRateWihStart("tps", start)

	require.EqualValues(t, 0, rate.export().Rate)
	measure(rate)

	rate.maybeRotateAsOf(start.Add(1100 * time.Millisecond))

	require.EqualValues(t, 100, rate.export().Rate)

	for i := 1; i < 10; i++ {
		rate.maybeRotateAsOf(start.Add(time.Duration(i) * time.Second))
	}

	require.Condition(t, func() bool {
		return rate.export().Rate < 100
	}, "rate did not decay")
}

func TestRate_MeasureSingleValue(t *testing.T) {
	testRateMeasure(t, func(rate *Rate) {
		rate.Measure(100)
	})
}

func TestRate_MeasureLoop(t *testing.T) {
	testRateMeasure(t, func(rate *Rate) {
		for i := 0; i < 100; i++ {
			rate.Measure(1)
		}
	})
}
