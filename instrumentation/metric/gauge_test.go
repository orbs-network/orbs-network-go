// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGauge_Add(t *testing.T) {
	g := Gauge{}
	g.AddUint32(10)

	require.EqualValues(t, 10, g.Value(), "gauge value differed from expected")
}

func TestGauge_Inc(t *testing.T) {
	g := Gauge{}
	g.Inc()

	require.EqualValues(t, 1, g.Value(), "gauge value differed from expected")
}

func TestGauge_Dec(t *testing.T) {
	g := Gauge{}
	g.Inc()
	g.Dec()

	require.EqualValues(t, 0, g.Value(), "gauge value differed from expected")
}

func TestGauge_SubUint32(t *testing.T) {
	g := Gauge{}
	g.AddUint32(10)
	g.SubUint32(10)

	require.EqualValues(t, 0, g.Value(), "gauge value differed from expected")
}

func TestGauge_Update(t *testing.T) {
	g := Gauge{}
	g.Update(123)

	require.EqualValues(t, 123, g.Value(), "gauge value differed from expected")
}

func TestGauge_UpdateUInt32(t *testing.T) {
	g := Gauge{}
	g.Update(321)

	require.EqualValues(t, 321, g.Value(), "gauge value differed from expected")
}
