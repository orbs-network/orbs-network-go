package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadKeysMissingKey(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newStateStorageDriver(1)
		d.commitValuePairs(ctx, "fooContract", "fooKey", "fooValue")

		value, err := d.readSingleKey(ctx, "fooContract", "someKey")
		require.NoError(t, err, "unexpected error")
		require.Equal(t, []byte{}, value, "expected zero value but received %v", value)
	})
}

func TestReadKeysReturnsCommittedValue(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		value := "bar"
		key := "foo"
		contract := "some-contract"

		d := newStateStorageDriver(1)
		d.commitValuePairs(ctx, contract, key, value, "someOtherKey", value)

		output, err := d.readSingleKey(ctx, contract, key)
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, value, output, "unexpected return value")
	})
}

func TestReadKeysBatch(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newStateStorageDriver(1)

		d.commitValuePairs(ctx, "contract", "key1", "bar1", "key2", "bar2", "key3", "bar3", "key4", "bar4", "key5", "bar5")

		output, err := d.readKeys(ctx, "contract", "key1", "key22", "key5", "key3", "key6")
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

		d := newStateStorageDriver(5)

		d.commitValuePairs(ctx, "contract1", key, v1)
		d.commitValuePairs(ctx, "contract2", key, v2)

		output, err := d.readSingleKey(ctx, "contract1", key)
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, v1, output, "read value %v when expecting %v", output, v1)

		output2, err2 := d.readSingleKey(ctx, "contract2", key)
		require.NoError(t, err2, "unexpected error")
		require.EqualValues(t, v2, output2, "read value %v when expecting %v", output, v1)
	})
}

func TestReadKeysInPastBlockHeights(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		key := "foo"
		v1, v2 := "bar", "bar2"

		d := newStateStorageDriver(5)
		d.commitValuePairsAtHeight(ctx, 1, "contract", key, v1)
		d.commitValuePairsAtHeight(ctx, 2, "contract", key, v2)

		historical, err := d.readSingleKeyFromRevision(ctx, 1, "contract", key)
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, v1, historical, "read value %v when expecting %v", historical, v1)

		current, err := d.readSingleKeyFromRevision(ctx, 2, "contract", key)
		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, v1, historical, "read value %v when expecting %v", current, v2)
	})
}

func TestReadKeysOutsideSupportedBlockRetention(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		key := "foo"

		d := newStateStorageDriver(1)
		d.commitValuePairsAtHeight(ctx, 1, "contract", key, "bar")
		d.commitValuePairsAtHeight(ctx, 2, "contract", key, "foo")

		output, err := d.readSingleKeyFromRevision(ctx, 1, "contract", key)
		require.Error(t, err, "expected an error to occur")
		require.Nil(t, output, "expected no result")
	})
}
