//+build !race
// Histogram metrics are disabled in race mode

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestDirectTransport_UpdatesRoundtripMetric(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2)
		defer shutdownAll(ctx, node1, node2)

		topology := aTopologyContaining(node1, node2)
		node1.transport.UpdateTopology(ctx, topology)
		node2.transport.UpdateTopology(ctx, topology)

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1, node2)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1, node2)

		roundtrip1to2 := node1.metricRegistry.Get(fmt.Sprintf("Gossip.OutgoingConnection.Roundtrip.%s", node2.address.String()[:6]))
		roundtrip2to1 := node2.metricRegistry.Get(fmt.Sprintf("Gossip.OutgoingConnection.Roundtrip.%s", node1.address.String()[:6]))

		for i := 1; i <= 5; i++ {
			node1.requireSendsSuccessfullyTo(t, ctx, node2)
			node2.requireSendsSuccessfullyTo(t, ctx, node1)

			require.True(t, test.Eventually(1*time.Second, func() bool {
				matched, err := regexp.MatchString(fmt.Sprintf("samples=%d", i), roundtrip1to2.String())
				require.NoError(t, err)
				return matched
			}), "expected 1->2 roundtrip metric to update")

			require.True(t, test.Eventually(1*time.Second, func() bool {
				matched, err := regexp.MatchString(fmt.Sprintf("samples=%d", i), roundtrip2to1.String())
				require.NoError(t, err)
				return matched
			}), "expected 2->1 roundtrip metric to update")
		}
	})
}
