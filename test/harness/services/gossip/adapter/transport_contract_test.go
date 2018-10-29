package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestContract_SendBroadcast(t *testing.T) {
	t.Run("TamperingTransport", broadcastTest(aTamperingTransport))
	t.Run("DirectTransport", broadcastTest(aDirectTransport))
}

func TestContract_SendToList(t *testing.T) {
	t.Skipf("TODO implement")
}

func TestContract_SendToAllButList(t *testing.T) {
	t.Skipf("TODO implement")
}

func broadcastTest(makeContext func(ctx context.Context) *transportContractContext) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		test.WithContext(func(ctx context.Context) {
			c := makeContext(ctx)

			data := &adapter.TransportData{
				SenderPublicKey: c.publicKeys[3],
				RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
				Payloads:        [][]byte{{0x71, 0x72, 0x73}},
			}

			c.listeners[0].expectReceive(data.Payloads)
			c.listeners[1].expectReceive(data.Payloads)
			c.listeners[2].expectReceive(data.Payloads)
			c.listeners[3].expectNotReceive()

			c.transports[3].Send(ctx, data)
			c.verify(t)
		})
	}
}

type transportContractContext struct {
	publicKeys []primitives.Ed25519PublicKey
	transports []adapter.Transport
	listeners  []*mockListener
}

func aTamperingTransport(ctx context.Context) *transportContractContext {
	res := &transportContractContext{}
	transport := NewTamperingTransport(log.GetLogger(log.String("adapter", "transport")))
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

func aDirectTransport(ctx context.Context) *transportContractContext {
	res := &transportContractContext{}

	// randomize listen port between tests to reduce flakiness and chances of listening clashes
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	firstRandomPort := 20000 + r.Intn(40000)

	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < 4; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(uint16(firstRandomPort+i), "127.0.0.1")
		res.publicKeys = append(res.publicKeys, publicKey)
	}

	configs := []config.GossipTransportConfig{
		config.ForGossipAdapterTests(res.publicKeys[0], uint16(firstRandomPort+0), gossipPeers),
		config.ForGossipAdapterTests(res.publicKeys[1], uint16(firstRandomPort+1), gossipPeers),
		config.ForGossipAdapterTests(res.publicKeys[2], uint16(firstRandomPort+2), gossipPeers),
		config.ForGossipAdapterTests(res.publicKeys[3], uint16(firstRandomPort+3), gossipPeers),
	}

	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	res.transports = []adapter.Transport{
		adapter.NewDirectTransport(ctx, configs[0], logger),
		adapter.NewDirectTransport(ctx, configs[1], logger),
		adapter.NewDirectTransport(ctx, configs[2], logger),
		adapter.NewDirectTransport(ctx, configs[3], logger),
	}
	res.listeners = []*mockListener{
		listenTo(res.transports[0], res.publicKeys[0]),
		listenTo(res.transports[1], res.publicKeys[1]),
		listenTo(res.transports[2], res.publicKeys[2]),
		listenTo(res.transports[3], res.publicKeys[3]),
	}

	// TODO: improve this, we need some time until everybody connects to everybody else
	// TODO: maybe add an adapter function to check how many active outgoing connections we have
	// @electricmonk proposal: Adapter could take a ConnectionListener that gets notified on connects/disconnects, and the test could provide such a listener to block until the desired number of connections has been reached
	time.Sleep(2 * configs[0].GossipConnectionKeepAliveInterval())

	return res
}

func (c *transportContractContext) verify(t *testing.T) {
	for _, mockListener := range c.listeners {
		// TODO: reduce eventually timeout to test.EVENTUALLY_ADAPTER_TIMEOUT once we remove memberlist
		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, mockListener))
	}
}
