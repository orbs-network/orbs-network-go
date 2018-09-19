package adapter

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContract_SendBroadcast(t *testing.T) {

	t.Run("TamperingTransport", broadcastTest(aTamperingTransport))
	t.Run("MemberlistTransport", broadcastTest(aMemberlistTransport))
}

func TestContract_SendToList(t *testing.T) {
	t.Skipf("TODO implement")
}

func TestContract_SendToAllButList(t *testing.T) {
	t.Skipf("TODO implement")
}

func broadcastTest(makeContext func() *transportContractContext) func(*testing.T) {

	return func(t *testing.T) {
		t.Parallel()
		c := makeContext()

		data := &adapter.TransportData{
			SenderPublicKey: c.publicKeys[3],
			RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
			Payloads:        [][]byte{{0x71, 0x72, 0x73}},
		}

		c.listeners[0].expectReceive(data.Payloads)
		c.listeners[1].expectReceive(data.Payloads)
		c.listeners[2].expectReceive(data.Payloads)
		c.listeners[3].expectNotReceive()

		c.transports[3].Send(data)
		c.verify(t)
	}
}

type transportContractContext struct {
	publicKeys []primitives.Ed25519PublicKey
	transports []adapter.Transport
	listeners  []*mockListener
}

func aTamperingTransport() *transportContractContext {
	res := &transportContractContext{}
	transport := NewTamperingTransport()
	res.publicKeys = []primitives.Ed25519PublicKey{{0x01}, {0x02}, {0x03}, {0x04}}
	res.transports = []adapter.Transport{transport, transport, transport, transport}
	res.listeners = []*mockListener{
		listenTo(res.transports[0], res.publicKeys[0]),
		listenTo(res.transports[1], res.publicKeys[1]),
		listenTo(res.transports[2], res.publicKeys[2]),
		listenTo(res.transports[3], res.publicKeys[3]),
	}
	return res
}

func aMemberlistTransport() *transportContractContext {
	logger := log.GetLogger()
	res := &transportContractContext{}
	res.publicKeys = []primitives.Ed25519PublicKey{{0x01}, {0x02}, {0x03}, {0x04}}
	configs := []adapter.MemberlistGossipConfig{
		{res.publicKeys[0], 60001, []string{"127.0.0.1:60002", "127.0.0.1:60003", "127.0.0.1:60004"}},
		{res.publicKeys[1], 60002, []string{"127.0.0.1:60001", "127.0.0.1:60003", "127.0.0.1:60004"}},
		{res.publicKeys[2], 60003, []string{"127.0.0.1:60001", "127.0.0.1:60002", "127.0.0.1:60004"}},
		{res.publicKeys[3], 60004, []string{"127.0.0.1:60001", "127.0.0.1:60002", "127.0.0.1:60003"}},
	}
	res.transports = []adapter.Transport{
		adapter.NewMemberlistTransport(configs[0], logger),
		adapter.NewMemberlistTransport(configs[1], logger),
		adapter.NewMemberlistTransport(configs[2], logger),
		adapter.NewMemberlistTransport(configs[3], logger),
	}
	res.listeners = []*mockListener{
		listenTo(res.transports[0], res.publicKeys[0]),
		listenTo(res.transports[1], res.publicKeys[1]),
		listenTo(res.transports[2], res.publicKeys[2]),
		listenTo(res.transports[3], res.publicKeys[3]),
	}
	return res
}

func (c *transportContractContext) verify(t *testing.T) {
	for _, mockListener := range c.listeners {
		// TODO: reduce eventually timeout to test.EVENTUALLY_ADAPTER_TIMEOUT once we remove memberlist
		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, mockListener))
	}
}
