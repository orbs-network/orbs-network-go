// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)


// The metrics-node (METRICS_NODE_INDEX in harness) did not load with blocks file and requires to sync.
// Eventually, after fully syncing this node will act as leader and propose blocks
func TestBlockSyncRecover(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	h := NewAppHarness()

	if h.envSupportsTestingFileAssets() {
		t.Skip("Running against remote network - skipping")
	}

	targetBlockHeight := CannedBlocksFileMinHeight + 20
	waitingDuration := 30*time.Second
	h.WaitUntilReachBlockHeight(t, CannedBlocksFileMinHeight, waitingDuration)

	var blockHeight primitives.BlockHeight
	test.Eventually(30*time.Second, func() bool {
		blockHeight = h.GetBlockHeight()
		return blockHeight >= targetBlockHeight
	})
	require.GreaterOrEqual(t, uint64(blockHeight), uint64(targetBlockHeight), "expected node in e2e network to sync and start closing blocks at block height greater than init blocks file")
}
