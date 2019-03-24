// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
		d := NewStateStorageDriver(1)

		root, err := d.service.GetStateHash(ctx, &services.GetStateHashInput{})
		require.NoError(t, err, "unexpected error")
		require.NotEqual(t, len(root.StateMerkleRootHash), 0, "uninitialized hash code result")
	})
}

func TestGetStateHashFutureHeightWithinGrace(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newStateStorageDriverWithGrace(1, 1, 1)

		output, err := d.service.GetStateHash(ctx, &services.GetStateHashInput{BlockHeight: 1})
		require.EqualError(t, errors.Cause(err), "context deadline exceeded", "expected timeout error")
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
		d := NewStateStorageDriver(1)

		root1, err := d.service.GetStateHash(ctx, &services.GetStateHashInput{})
		require.NoError(t, err, "unexpected error")

		d.CommitValuePairs(ctx, "foo", "bar", "baz")

		root2, err1 := d.service.GetStateHash(ctx, &services.GetStateHashInput{BlockHeight: primitives.BlockHeight(1)})
		require.NoError(t, err1, "unexpected error")
		require.NotEqual(t, len(root2.StateMerkleRootHash), 0, "uninitialized hash code result after state change")
		require.NotEqual(t, root2.StateMerkleRootHash, root1.StateMerkleRootHash, "merkle root identical after state change")
	})
}
