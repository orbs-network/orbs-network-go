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

func TestPersistStateToStorage(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)

		contract1 := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "v1").WithStringRecord("key2", "v2").Build()
		contract2 := builders.ContractStateDiff().WithContractName("contract2").WithStringRecord("key1", "v3").Build()

		d.CommitStateDiff(ctx, CommitStateDiff().WithBlockHeight(1).WithDiff(contract1).WithDiff(contract2).Build())

		output, err := d.ReadSingleKey(ctx, "contract1", "key1")
		require.NoError(t, err)
		require.EqualValues(t, "v1", output, "unexpected value read from storage")
		output2, err := d.ReadSingleKey(ctx, "contract1", "key2")
		require.NoError(t, err)
		require.EqualValues(t, "v2", output2, "unexpected value read from storage")
		output3, err := d.ReadSingleKey(ctx, "contract2", "key1")
		require.NoError(t, err)
		require.EqualValues(t, "v3", output3, "unexpected value read from storage")
	})
}

func TestNonConsecutiveBlockHeights(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)

		registerContractDiff := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "whatever").Build()
		d.service.CommitStateDiff(ctx, CommitStateDiff().WithBlockHeight(1).WithDiff(registerContractDiff).Build())

		diff := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "whatever").Build()
		result, err := d.service.CommitStateDiff(ctx, CommitStateDiff().WithBlockHeight(3).WithDiff(diff).Build())

		require.NoError(t, err)
		require.EqualValues(t, 2, result.NextDesiredBlockHeight, "unexpected NextDesiredBlockHeight")

		_, err = d.ReadSingleKey(ctx, "contract1", "key1")
		require.NoError(t, err)
	})
}

func TestCommitPastBlockHeights(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := NewStateStorageDriver(1)
		v1 := "v1"
		v2 := "v2"

		d.CommitValuePairsAtHeight(ctx, 1, "c1", "key1", v1)
		d.CommitValuePairsAtHeight(ctx, 2, "c1", "key1", v2)

		result, err := d.CommitValuePairsAtHeight(ctx, 1, "c1", "key1", "v3", "key3", "v3")
		require.NoError(t, err)
		require.EqualValues(t, 3, result.NextDesiredBlockHeight, "unexpected NextDesiredBlockHeight")

		output, err := d.ReadSingleKeyFromRevision(ctx, 2, "c1", "key1")
		require.NoError(t, err)
		require.EqualValues(t, v2, output, "unexpected value read")
		output2, err := d.ReadSingleKeyFromRevision(ctx, 2, "c1", "key3")
		require.NoError(t, err)
		require.EqualValues(t, []byte{}, output2, "unexpected value read")
	})
}
