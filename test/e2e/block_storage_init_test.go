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
	const expectedBlocks = 500
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {
		h := newHarness()
		require.True(t, test.Eventually(2*time.Second, func() bool {
			blockHeight := h.getMetrics()["BlockStorage.BlockHeight"]["Value"].(float64)
			return blockHeight >= expectedBlocks
		}), "expected e2e network to launch with %v blocks", expectedBlocks)
	})
}
