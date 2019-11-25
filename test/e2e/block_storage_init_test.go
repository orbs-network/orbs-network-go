// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestInitialBlockHeight(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	h := NewAppHarness()

	if h.envSupportsLoadingWithCannedBlocksFile() {
		t.Skip("Running against remote network - skipping")
	}

	h.WaitUntilTransactionPoolIsReady(t)

	var blockHeight uint64
	test.Eventually(5*time.Second, func() bool {
		blockHeight = uint64(h.GetMetrics()["BlockStorage.BlockHeight"]["Value"].(float64))
		return blockHeight >= CannedBlocksFileMinHeight
	})
	require.GreaterOrEqual(t, blockHeight, uint64(CannedBlocksFileMinHeight), "expected e2e network to start closing blocks at block height greater than init blocks file")
}
