// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadKeysMissingKey(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)
		d.CommitValuePairs(ctx, "fooContract", "fooKey", "fooValue")

		value, err := d.ReadSingleKey(ctx, "fooContract", "someKey")
		require.NoError(t, err, "unexpected error")
		require.Equal(t, []byte{}, value, "expected zero value but received %v", value)
	})
}

func TestReadKeysReturnsCommittedValue(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		value := "bar"
		key := "foo"
		contract := "some-contract"

		d := NewStateStorageDriver(1)
		d.CommitValuePairs(ctx, contract, key, value, "someOtherKey", value)

		output, err := d.ReadSingleKey(ctx, contract, key)
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, value, output, "unexpected return value")
	})
}

func TestReadKeysBatch(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)

		d.CommitValuePairs(ctx, "contract", "key1", "bar1", "key2", "bar2", "key3", "bar3", "key4", "bar4", "key5", "bar5")

		output, err := d.ReadKeys(ctx, "contract", "key1", "key22", "key5", "key3", "key6")
		require.NoError(t, err, "unexpected error")

		require.Len(t, output, 5, "response length does not match number of keys in request")
		require.EqualValues(t, *output[0], keyValue{"key1", []byte("bar1")}, "unexpected output at position 0")
		require.EqualValues(t, *output[1], keyValue{"key22", []byte{}}, "unexpected output at position 1")
		require.EqualValues(t, *output[2], keyValue{"key5", []byte("bar5")}, "unexpected output at position 2")
		require.EqualValues(t, *output[3], keyValue{"key3", []byte("bar3")}, "unexpected output at position 3")
		require.EqualValues(t, *output[4], keyValue{"key6", []byte{}}, "unexpected output at position 4")
	})
}

func TestReadSameKeyFromDifferentContracts(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		key := "foo"
		v1, v2 := "bar", "bar2"

		d := NewStateStorageDriver(5)

		d.CommitValuePairs(ctx, "contract1", key, v1)
		d.CommitValuePairs(ctx, "contract2", key, v2)

		output, err := d.ReadSingleKey(ctx, "contract1", key)
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, v1, output, "read value %v when expecting %v", output, v1)

		output2, err2 := d.ReadSingleKey(ctx, "contract2", key)
		require.NoError(t, err2, "unexpected error")
		require.EqualValues(t, v2, output2, "read value %v when expecting %v", output, v1)
	})
}

func TestReadKeysInPastBlockHeights(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		key := "foo"
		v1, v2 := "bar", "bar2"

		d := NewStateStorageDriver(5)
		d.CommitValuePairsAtHeight(ctx, 1, "contract", key, v1)
		d.CommitValuePairsAtHeight(ctx, 2, "contract", key, v2)

		historical, err := d.ReadSingleKeyFromRevision(ctx, 1, "contract", key)
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, v1, historical, "read value %v when expecting %v", historical, v1)

		current, err := d.ReadSingleKeyFromRevision(ctx, 2, "contract", key)
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, v1, historical, "read value %v when expecting %v", current, v2)
	})
}

func TestReadKeysOutsideSupportedBlockRetention(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		key := "foo"

		d := NewStateStorageDriver(1)
		d.CommitValuePairsAtHeight(ctx, 1, "contract", key, "bar")
		d.CommitValuePairsAtHeight(ctx, 2, "contract", key, "foo")

		output, err := d.ReadSingleKeyFromRevision(ctx, 1, "contract", key)
		require.Error(t, err, "expected an error to occur")
		require.Nil(t, output, "expected no result")
	})
}

func TestReadKeysObservesWriteOrder(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		key := "foo"

		d := NewStateStorageDriver(1)
		d.CommitValuePairsAtHeight(ctx, 1, "c", key, "bar", key, "baz")

		output, err := d.ReadSingleKeyFromRevision(ctx, 1, "c", key)
		require.NoError(t, err)
		require.EqualValues(t, "baz", output, "expected no result")
	})
}
