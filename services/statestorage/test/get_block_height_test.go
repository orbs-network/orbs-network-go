// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitToZero(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)
		height, timestamp, err := d.GetBlockHeightAndTimestamp(ctx)

		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, 0, height, "unexpected height")
		require.EqualValues(t, 0, timestamp, "unexpected timestamp")
	})
}

func TestReflectsSuccessfulCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)
		heightBefore, _, _ := d.GetBlockHeightAndTimestamp(ctx)
		d.service.CommitStateDiff(ctx, CommitStateDiff().WithBlockHeight(1).WithBlockTimestamp(6579).WithDiff(builders.ContractStateDiff().Build()).Build())
		heightAfter, timestampAfter, err := d.GetBlockHeightAndTimestamp(ctx)

		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, heightBefore+1, heightAfter, "unexpected height")
		require.EqualValues(t, 6579, timestampAfter, "unexpected timestamp")
	})
}

func TestIgnoreFailedCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)
		stateDiff := builders.ContractStateDiff().Build()
		d.service.CommitStateDiff(ctx, CommitStateDiff().WithBlockHeight(1).WithDiff(stateDiff).Build())
		d.service.CommitStateDiff(ctx, CommitStateDiff().WithBlockHeight(2).WithDiff(stateDiff).Build())
		heightBefore, _, _ := d.GetBlockHeightAndTimestamp(ctx)
		d.service.CommitStateDiff(ctx, CommitStateDiff().WithBlockHeight(1).WithDiff(stateDiff).Build())
		heightAfter, _, err := d.GetBlockHeightAndTimestamp(ctx)

		require.NoError(t, err, "unexpected error")
		require.EqualValues(t, heightBefore, heightAfter, "unexpected height")
	})
}
