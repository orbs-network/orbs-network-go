package e2e

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitialBlockHeight(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := newHarness()
		blockHeight := h.getMetrics()["BlockStorage.BlockHeight"]["Value"].(float64)
		require.Truef(t, blockHeight >= 0x5a, "expected e2e network to launch with some blocks, found only %d", blockHeight)
	})
}
