// +build memoryleak

package acceptance

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
)

// as we are using a build flag, and we want to avoid logging in the stress test
// as the harness will cache them because of t.Log, we have this conditional compilation for creating the harness
func getStressTestHarness() *networkHarnessBuilder {
	return newHarness().WithLogFilters(log.DiscardAll())
}
