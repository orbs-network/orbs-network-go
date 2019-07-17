// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDirectTransport_HandlesStartupWithEmptyPeerList(t *testing.T) {
	// High value to disable keep alive

	cfg := config.ForDirectTransportTests(make(map[string]config.GossipPeer), 20*time.Hour, 1*time.Second)
	test.WithContext(func(ctx context.Context) {
		transport := NewDirectTransport(ctx, cfg, log.DefaultTestingLogger(t), metric.NewRegistry())
		require.True(t, test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
			return transport.IsServerListening()
		}), "server did not start")
	})
}

func TestDirectTransport_SupportsAddingPeersInRuntime(t *testing.T) {
	// High value to disable keep alive

	cfg := config.ForDirectTransportTests(make(map[string]config.GossipPeer), 20*time.Hour, 1*time.Second)
	test.WithContext(func(ctx context.Context) {
		node1 := NewDirectTransport(ctx, cfg, log.DefaultTestingLogger(t), metric.NewRegistry())
		node2 := NewDirectTransport(ctx, cfg, log.DefaultTestingLogger(t), metric.NewRegistry())
		address1 := keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()
		address2 := keys.EcdsaSecp256K1KeyPairForTests(2).NodeAddress()
		l1 := &testkit.MockTransportListener{}
		l2 := &testkit.MockTransportListener{}
		node1.RegisterListener(l1, address1)
		node1.RegisterListener(l2, address2)

		require.True(t, test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
			return node1.IsServerListening() && node2.IsServerListening()
		}), "server did not start")

		node1.AddPeer(ctx, address2, config.NewHardCodedGossipPeer(node2.serverPort, "127.0.0.1"))
		node2.AddPeer(ctx, address1, config.NewHardCodedGossipPeer(node1.serverPort, "127.0.0.1"))

		require.True(t, test.Eventually(HARNESS_OUTGOING_CONNECTIONS_INIT_TIMEOUT, func() bool {
			return node1.safeLenOfOutgoingPeerQueues() > 0 && node2.safeLenOfOutgoingPeerQueues() > 0
		}), "expected all outgoing queues to become enabled after successfully connecting to added peers")

		header := (&gossipmessages.HeaderBuilder{
			Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
			RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		}).Build()

		message := &gossipmessages.LeanHelixMessage{
			Content: []byte{},
		}

		payloads := [][]byte{header.Raw(), message.Content}

		l2.ExpectReceive(payloads)
		require.NoError(t, node1.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      address1,
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{address2},
			Payloads:               payloads,
		}))

		l1.ExpectReceive(payloads)
		require.NoError(t, node2.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      address2,
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{address1},
			Payloads:               payloads,
		}))
	})
}
