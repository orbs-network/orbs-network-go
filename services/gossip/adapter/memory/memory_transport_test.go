// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package memory

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMemoryTransport_PropagatesTracingContext(t *testing.T) {
	with.Concurrency(t, func(parentContext context.Context, harness *with.ConcurrencyHarness) {
		address := primitives.NodeAddress{0x01}
		transport := NewTransport(parentContext, harness.Logger, makeNetwork(address))
		harness.Supervise(transport)
		defer transport.GracefulShutdown(parentContext)

		listener := testkit.ListenTo(transport, address)

		childContext, cancel := context.WithCancel(parentContext) // this is required so that the parent context does not get polluted
		defer cancel()

		contextWithTrace := trace.NewContext(childContext, "foo")
		originalTracingContext, _ := trace.FromContext(contextWithTrace)

		listener.ExpectTracingContextToPropagate(t, originalTracingContext)

		_ = transport.Send(contextWithTrace, &adapter.TransportData{
			SenderNodeAddress: primitives.NodeAddress{0x02},
			RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		})

		require.NoError(t, test.EventuallyVerify(100*time.Millisecond, listener))
	})
}

func TestMemoryTransport_SendIsAsynchronous_NoListener(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		address := primitives.NodeAddress{0x01}
		transport := NewTransport(ctx, harness.Logger, makeNetwork(address))
		harness.Supervise(transport)
		defer transport.GracefulShutdown(ctx)

		// sending without a listener - nobody is receiving
		transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress: primitives.NodeAddress{0x02},
			RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		})

	})
}

func TestMemoryTransport_SendIsAsynchronous_BlockedListener(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		address := primitives.NodeAddress{0x01}
		transport := NewTransport(ctx, harness.Logger, makeNetwork(address))
		harness.Supervise(transport)
		defer transport.GracefulShutdown(ctx)

		listener := testkit.ListenTo(transport, address)
		listener.BlockReceive()

		for i := 0; i < 2; i++ {
			transport.Send(ctx, &adapter.TransportData{
				SenderNodeAddress: primitives.NodeAddress{0x02},
				RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
			})
		}

	})
}

func TestMemoryTransport_DoesNotGetStuckWhenSendBufferIsFull(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		address := primitives.NodeAddress{0x01}
		transport := NewTransport(ctx, harness.Logger, makeNetwork(address))
		harness.Supervise(transport)
		defer transport.GracefulShutdown(ctx)

		harness.AllowErrorsMatching("memory transport send buffer is full")

		listener := testkit.ListenTo(transport, address)
		listener.BlockReceive()

		// log error "memory transport send buffer is full" is expected in this test
		for i := 0; i < SEND_QUEUE_MAX_MESSAGES+10; i++ {
			transport.Send(ctx, &adapter.TransportData{
				SenderNodeAddress: primitives.NodeAddress{0x02},
				RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
			})
		}

	})
}

func makeNetwork(addresses ...primitives.NodeAddress) map[string]config.ValidatorNode {
	genesisValidatorNodes := make(map[string]config.ValidatorNode)
	for _, address := range addresses {
		genesisValidatorNodes[address.KeyForMap()] = config.NewHardCodedValidatorNode(address)
	}
	return genesisValidatorNodes
}
