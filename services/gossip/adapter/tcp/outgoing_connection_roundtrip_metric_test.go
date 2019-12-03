//+build !race
// Histogram metrics are disabled in race mode

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestOutgoingConnection_RoundtripMetricReflectsLatency(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			server := newServerStub(t)
			defer server.Close()

			client := server.createClientAndConnect(ctx, t, parent.Logger, 10*time.Millisecond)
			waitForQueueEnabled(t, client)

			client.addDataToOutgoingPeerQueue(ctx, &adapter.TransportData{
				SenderNodeAddress:      []byte{},
				RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
				RecipientNodeAddresses: []primitives.NodeAddress{[]byte{}},
				Payloads:               aMessage(),
			})

			time.Sleep(200 * time.Millisecond) // create an artificial latency

			// No acks
			matched, err := regexp.MatchString(fmt.Sprintf("samples=0"), client.roundtripMetric.String())
			require.NoError(t, err)
			require.True(t, matched, "expected 0 samples in the roundtrip metric")

			// Send ack
			_, err = server.conn.Write(ACK_BUFFER)
			require.NoError(t, err, "couldn't write ack")

			// see that the metric has updated
			require.True(t, test.Eventually(1*time.Second, func() bool {
				matched, err := regexp.MatchString(fmt.Sprintf("samples=1"), client.roundtripMetric.String())
				require.NoError(t, err)
				return matched
			}), "expected roundtrip metric to update")
			require.True(t, test.Eventually(1*time.Second, func() bool {
				matched, err := regexp.MatchString(fmt.Sprintf("max=2\\d\\d\\."), client.roundtripMetric.String())
				require.NoError(t, err)
				return matched
			}), "expected roundtrip metric to be in the range of 2XX ms")
		})
	})
}

func TestOutgoingConnection_SendsMessagesWhenAcksAreDelayed(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			server := newServerStub(t)
			defer server.Close()

			client := server.createClientAndConnect(ctx, t, parent.Logger, 10*time.Millisecond)
			waitForQueueEnabled(t, client)

			const iterations = 10 * MaxTransmissionTimeQueueSize // a lot more than the queue size

			for i := 0; i < iterations; i++ {
				client.addDataToOutgoingPeerQueue(ctx, &adapter.TransportData{
					SenderNodeAddress:      []byte{},
					RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
					RecipientNodeAddresses: []primitives.NodeAddress{[]byte{}},
					Payloads:               aMessage(),
				})
				_ = server.readMessage(ctx)
			}

			// No acks
			matched, err := regexp.MatchString(fmt.Sprintf("samples=0"), client.roundtripMetric.String())
			require.NoError(t, err)
			require.True(t, matched, "expected 0 samples in the roundtrip metric")

			// send back acks
			for i := 0; i < iterations; i++ {
				_, err := server.conn.Write(ACK_BUFFER)
				require.NoError(t, err, "couldn't write ack")
			}

			// see that the metric has updated
			require.True(t, test.Eventually(1*time.Second, func() bool {
				matched, err := regexp.MatchString(fmt.Sprintf("samples=%d", MaxTransmissionTimeQueueSize), client.roundtripMetric.String())
				require.NoError(t, err)
				return matched
			}), "expected roundtrip metric to update exactly MaxTransmissionTimeQueueSize times")
		})
	})
}
