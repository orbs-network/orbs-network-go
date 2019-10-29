// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/tcp"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestContract_SendBroadcast(t *testing.T) {
	t.Run("TCP_DirectTransport", broadcastTest(aDirectTransport))
	t.Run("MemoryTransport", broadcastTest(aMemoryTransport))
}

func TestContract_SendToList(t *testing.T) {
	t.Run("TCP_DirectTransport", sendToListTest(aDirectTransport))
	t.Run("MemoryTransport", sendToListTest(aMemoryTransport))
}

func TestContract_SendToAllButList(t *testing.T) {
	t.Skipf("implement") // TODO(v1)
}

func broadcastTest(makeContext func(ctx context.Context, harness *with.ConcurrencyHarness) *transportContractContext) func(*testing.T) {
	return func(t *testing.T) {
		with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
			c := makeContext(ctx, harness)
			defer c.shutdownAll(ctx)

			data := &adapter.TransportData{
				SenderNodeAddress: c.nodeAddresses[3],
				RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
				Payloads:          [][]byte{{0x71, 0x72, 0x73}},
			}

			c.listeners[0].ExpectReceive(data.Payloads)
			c.listeners[1].ExpectReceive(data.Payloads)
			c.listeners[2].ExpectReceive(data.Payloads)
			c.listeners[3].ExpectNotReceive()

			require.True(t, c.eventuallySendAndVerify(ctx, c.transports[3], data))
		})
	}
}

func sendToListTest(makeContext func(ctx context.Context, harness *with.ConcurrencyHarness) *transportContractContext) func(*testing.T) {
	return func(t *testing.T) {
		with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
			c := makeContext(ctx, harness)
			defer c.shutdownAll(ctx)

			data := &adapter.TransportData{
				SenderNodeAddress:      c.nodeAddresses[3],
				RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
				RecipientNodeAddresses: []primitives.NodeAddress{c.nodeAddresses[1], c.nodeAddresses[2]},
				Payloads:               [][]byte{{0x81, 0x82, 0x83}},
			}

			c.listeners[0].ExpectNotReceive()
			c.listeners[1].ExpectReceive(data.Payloads)
			c.listeners[2].ExpectReceive(data.Payloads)
			c.listeners[3].ExpectNotReceive()

			require.True(t, c.eventuallySendAndVerify(ctx, c.transports[3], data))
		})
	}
}

type transportContractContext struct {
	nodeAddresses []primitives.NodeAddress
	transports    []adapter.Transport
	listeners     []*testkit.MockTransportListener
}

func aMemoryTransport(ctx context.Context, harness *with.ConcurrencyHarness) *transportContractContext {
	res := &transportContractContext{}
	res.nodeAddresses = []primitives.NodeAddress{{0x01}, {0x02}, {0x03}, {0x04}}

	genesisValidatorNodes := make(map[string]config.ValidatorNode)
	for _, address := range res.nodeAddresses {
		genesisValidatorNodes[address.KeyForMap()] = config.NewHardCodedValidatorNode(primitives.NodeAddress(address))
	}
	logger := harness.Logger.WithTags(log.String("adapter", "transport"))

	transport := memory.NewTransport(ctx, logger, genesisValidatorNodes)
	res.transports = []adapter.Transport{transport, transport, transport, transport}
	res.listeners = []*testkit.MockTransportListener{
		testkit.ListenTo(res.transports[0], res.nodeAddresses[0]),
		testkit.ListenTo(res.transports[1], res.nodeAddresses[1]),
		testkit.ListenTo(res.transports[2], res.nodeAddresses[2]),
		testkit.ListenTo(res.transports[3], res.nodeAddresses[3]),
	}

	harness.Supervise(transport)

	return res
}

func aDirectTransport(ctx context.Context, harness *with.ConcurrencyHarness) *transportContractContext {
	res := &transportContractContext{}

	for i := 0; i < 4; i++ {
		nodeAddress := keys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		res.nodeAddresses = append(res.nodeAddresses, nodeAddress)
	}

	configs := []config.GossipTransportConfig{
		config.ForGossipAdapterTests(res.nodeAddresses[0]),
		config.ForGossipAdapterTests(res.nodeAddresses[1]),
		config.ForGossipAdapterTests(res.nodeAddresses[2]),
		config.ForGossipAdapterTests(res.nodeAddresses[3]),
	}

	logger := harness.Logger.WithTags(log.String("adapter", "transport"))

	transports := []*tcp.DirectTransport{
		tcp.NewDirectTransport(ctx, configs[0], logger, metric.NewRegistry()),
		tcp.NewDirectTransport(ctx, configs[1], logger, metric.NewRegistry()),
		tcp.NewDirectTransport(ctx, configs[2], logger, metric.NewRegistry()),
		tcp.NewDirectTransport(ctx, configs[3], logger, metric.NewRegistry()),
	}

	test.Eventually(1*time.Second, func() bool {
		listening := true
		for _, t := range transports {
			listening = listening && t.IsServerListening()
		}
		return listening
	})

	res.listeners = []*testkit.MockTransportListener{
		testkit.ListenTo(transports[0], res.nodeAddresses[0]),
		testkit.ListenTo(transports[1], res.nodeAddresses[1]),
		testkit.ListenTo(transports[2], res.nodeAddresses[2]),
		testkit.ListenTo(transports[3], res.nodeAddresses[3]),
	}

	peers := make(tcp.GossipPeers)
	for i, transport := range transports {
		peers[res.nodeAddresses[i].KeyForMap()] = config.NewHardCodedGossipPeer(transport.GetServerPort(), "127.0.0.1", hex.EncodeToString(res.nodeAddresses[i]))
	}

	for _, t1 := range transports {
		t1.UpdateTopology(ctx, peers)
	}

	for _, t := range transports {
		harness.Supervise(t)

		res.transports = append(res.transports, t)
	}

	return res
}

// Continuously retry to send a message and verify mock listeners.
// When Transport.Send() is called we get no guarantee for delivery.
// the returned error is not intended to reflect success in neither sending or
// queuing of the message. error is returned only for internal configuration
// conflicts, namely, trying to send to an unknown recipient.
//
// Send() will not return an error if the connection is closed, not yet connected,
// if the buffer is overflowed, or for any other networking issue.
//
// For this reason we must re-Send() on every iteration of the verification loop.
// It is also the reason why the mock verification conditions must be
// tolerant to receiving the message more than once as it is likely
// some listeners will receive multiple transmissings of data
func (c *transportContractContext) eventuallySendAndVerify(ctx context.Context, sender adapter.Transport, data *adapter.TransportData) bool {
	cfg := config.ForGossipAdapterTests(nil)
	return test.Eventually(2*cfg.GossipNetworkTimeout(), func() bool {

		err := sender.Send(ctx, data) // try to resend
		if err != nil {               // does not indicate a failure to send, only on config issues
			return false
		}

		for _, mockListener := range c.listeners {
			if ok, _ := mockListener.Verify(); !ok {
				return false
			}
		}
		return true

	})
}

func (c *transportContractContext) shutdownAll(ctx context.Context) {
	for _, t := range c.transports {
		t.GracefulShutdown(ctx)
	}
}
