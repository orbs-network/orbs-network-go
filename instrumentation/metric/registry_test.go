// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInMemoryRegistry_Export(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		registry := NewRegistry()
		gauge := registry.NewGauge("hello.world")
		gauge.Add(1)

		export := registry.ExportAllNested(parent.Logger)

		firstLevel, found := export["hello"]
		require.True(t, found)
		secondLevel, found := firstLevel.(exportedMap)["world"]
		require.True(t, found)

		require.EqualValues(t, secondLevel.(int64), 1)
	})
}

func TestInMemoryRegistry_ExportThenGet(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		// prepare
		registry := NewRegistry()
		items := []struct {
			name  string
			value int64
		}{
			//		{"hello", 1},
			{"hello.world", 2},
			{"hello.world.my.Count", 3},
			{"block.storage.height", 4},
			{"block.storage.time", 5},
			{"TransactionPool.LastCommitted.TimeNano", 6},
		}
		for i := 0; i < len(items); i++ {
			createItem(registry, items[i].name, items[i].value)
		}
		// do
		data, _ := json.Marshal(registry.ExportAllNested(parent.Logger))
		export := make(exportedMap)
		err := json.Unmarshal(data, &export)
		require.NoError(t, err)

		// assert existing names
		for i := 0; i < len(items); i++ {
			val, found := export.GetAsInt(items[i].name)
			require.True(t, found)
			require.EqualValues(t, val, items[i].value)
		}

		// assert bad names
		badNames := []string{
			"world",
			"hello.me",
			"hello.world.2",
			"block.storage",
		}
		for i := 0; i < len(badNames); i++ {
			_, found := export.GetAsInt(badNames[i])
			require.False(t, found)
		}
	})
}

func TestInMemoryRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	gauge := registry.NewGauge("hello")
	gaugeFromReg := registry.Get("hello")

	require.Equal(t, gauge, gaugeFromReg)
	require.NotEmpty(t, registry.mu.metrics)
}

func TestInMemoryRegistry_Remove(t *testing.T) {
	registry := NewRegistry()
	gauge := registry.NewGauge("hello")
	registry.Remove(gauge)

	require.Empty(t, registry.mu.metrics)
}

func TestInMemoryRegistry_RemoveSameNameDifferentObject(t *testing.T) {
	registry := NewRegistry()
	gauge := registry.NewGauge("hello")
	otherGaugeSameName := newGauge("hello", "hello")
	registry.Remove(otherGaugeSameName)

	require.NotEmpty(t, registry.mu.metrics)
	gaugeFromReg := registry.Get("hello")
	require.Equal(t, gauge, gaugeFromReg)
}

func TestInMemoryRegistry_RegisterSameNamePanics(t *testing.T) {
	registry := NewRegistry()
	registry.NewGauge("hello")

	require.Panics(t, func() {
		registry.NewGauge("hello")
	}, "should have rejected same name")
}

func TestInMemoryRegistry_RegisterSameNameDiffTypePanics(t *testing.T) {
	registry := NewRegistry()
	registry.NewGauge("hello")

	require.Panics(t, func() {
		registry.NewText("hello")
	}, "should have rejected same name")
}

func createItem(registry Registry, name string, value int64) {
	gauge := registry.NewGauge(name)
	gauge.Add(value)
}
