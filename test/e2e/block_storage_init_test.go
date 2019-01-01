package e2e

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitialBlockHeight(t *testing.T) {
	const expectedBlocks = 500
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := newHarness()
		blockHeight := h.getMetrics()["BlockStorage.BlockHeight"]["Value"].(float64)
		require.Truef(t, blockHeight >= expectedBlocks, "expected e2e network to launch with %v blocks, found only %v", expectedBlocks, blockHeight)
	})
}
