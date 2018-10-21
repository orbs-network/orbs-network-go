package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetStateHashReturnsNonZeroValue(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newStateStorageDriver(1)

		root, err := d.service.GetStateHash(ctx, &services.GetStateHashInput{})
		require.NoError(t, err, "unexpected error")
		require.NotEqual(t, len(root.StateRootHash), 0, "uninitialized hash code result")
	})
}

func TestGetStateHashFutureHeightWithinGrace(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newStateStorageDriverWithGrace(1, 1, 1)

		output, err := d.service.GetStateHash(ctx, &services.GetStateHashInput{BlockHeight: 1})
		require.EqualError(t, errors.Cause(err), "timed out waiting for block at height 1", "expected timeout error")
		require.Nil(t, output, "expected nil output when timing out")
	})
}

func TestGetStateHashFutureHeightOutsideGrace(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newStateStorageDriverWithGrace(1, 1, 1)

		output, err := d.service.GetStateHash(ctx, &services.GetStateHashInput{BlockHeight: 2})
		require.EqualError(t, errors.Cause(err), "requested future block outside of grace range", "expected out of range error")
		require.Nil(t, output, "expected nil output when block height out of range")
	})
}

func TestGetStateHashMerkleRootChangesOnStateChange(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newStateStorageDriver(1)

		root1, err := d.service.GetStateHash(ctx, &services.GetStateHashInput{})
		require.NoError(t, err, "unexpected error")

		d.commitValuePairs(ctx, "foo", "bar", "baz")

		root2, err1 := d.service.GetStateHash(ctx, &services.GetStateHashInput{BlockHeight: primitives.BlockHeight(1)})
		require.NoError(t, err1, "unexpected error")
		require.NotEqual(t, len(root2.StateRootHash), 0, "uninitialized hash code result after state change")
		require.NotEqual(t, root2.StateRootHash, root1.StateRootHash, "merkle root identical after state change")
	})
}
