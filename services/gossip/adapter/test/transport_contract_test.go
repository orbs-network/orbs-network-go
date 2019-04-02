// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/tcp"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
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

func broadcastTest(makeContext func(ctx context.Context, tb testing.TB) *transportContractContext) func(*testing.T) {
	return func(t *testing.T) {
		test.WithContext(func(ctx context.Context) {
			c := makeContext(ctx, t)

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

func sendToListTest(makeContext func(ctx context.Context, tb testing.TB) *transportContractContext) func(*testing.T) {
	return func(t *testing.T) {
		test.WithContext(func(ctx context.Context) {
			c := makeContext(ctx, t)

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

func aMemoryTransport(ctx context.Context, tb testing.TB) *transportContractContext {
	res := &transportContractContext{}
	res.nodeAddresses = []primitives.NodeAddress{{0x01}, {0x02}, {0x03}, {0x04}}

	genesisValidatorNodes := make(map[string]config.ValidatorNode)
	for _, address := range res.nodeAddresses {
		genesisValidatorNodes[address.KeyForMap()] = config.NewHardCodedValidatorNode(primitives.NodeAddress(address))
	}

	logger := log.DefaultTestingLogger(tb).WithTags(log.String("adapter", "transport"))

	transport := memory.NewTransport(ctx, logger, genesisValidatorNodes)
	res.transports = []adapter.Transport{transport, transport, transport, transport}
	res.listeners = []*testkit.MockTransportListener{
		testkit.ListenTo(res.transports[0], res.nodeAddresses[0]),
		testkit.ListenTo(res.transports[1], res.nodeAddresses[1]),
		testkit.ListenTo(res.transports[2], res.nodeAddresses[2]),
		testkit.ListenTo(res.transports[3], res.nodeAddresses[3]),
	}

	return res
}

func aDirectTransport(ctx context.Context, tb testing.TB) *transportContractContext {
	res := &transportContractContext{}

	gossipPortByNodeIndex := []int{}
	gossipPeers := make(map[string]config.GossipPeer)

	for i := 0; i < 4; i++ {
		gossipPortByNodeIndex = append(gossipPortByNodeIndex, test.RandomPort())
		nodeAddress := keys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		gossipPeers[nodeAddress.KeyForMap()] = config.NewHardCodedGossipPeer(gossipPortByNodeIndex[i], "127.0.0.1")
		res.nodeAddresses = append(res.nodeAddresses, nodeAddress)
	}

	configs := []config.GossipTransportConfig{
		config.ForGossipAdapterTests(res.nodeAddresses[0], gossipPortByNodeIndex[0], gossipPeers),
		config.ForGossipAdapterTests(res.nodeAddresses[1], gossipPortByNodeIndex[1], gossipPeers),
		config.ForGossipAdapterTests(res.nodeAddresses[2], gossipPortByNodeIndex[2], gossipPeers),
		config.ForGossipAdapterTests(res.nodeAddresses[3], gossipPortByNodeIndex[3], gossipPeers),
	}

	logger := log.DefaultTestingLogger(tb)
	registry := metric.NewRegistry()

	res.transports = []adapter.Transport{
		tcp.NewDirectTransport(ctx, configs[0], logger, registry),
		tcp.NewDirectTransport(ctx, configs[1], logger, registry),
		tcp.NewDirectTransport(ctx, configs[2], logger, registry),
		tcp.NewDirectTransport(ctx, configs[3], logger, registry),
	}
	res.listeners = []*testkit.MockTransportListener{
		testkit.ListenTo(res.transports[0], res.nodeAddresses[0]),
		testkit.ListenTo(res.transports[1], res.nodeAddresses[1]),
		testkit.ListenTo(res.transports[2], res.nodeAddresses[2]),
		testkit.ListenTo(res.transports[3], res.nodeAddresses[3]),
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
	cfg := config.ForGossipAdapterTests(nil, 0, nil)
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
