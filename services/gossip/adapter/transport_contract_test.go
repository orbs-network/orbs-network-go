package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestContract_SendBroadcast(t *testing.T) {
	t.Run("DirectTransport", broadcastTest(aDirectTransport))
	t.Run("ChannelTransport", broadcastTest(aChannelTransport))
}

func TestContract_SendToList(t *testing.T) {
	t.Skipf("TODO implement")
}

func TestContract_SendToAllButList(t *testing.T) {
	t.Skipf("TODO implement")
}

func broadcastTest(makeContext func(ctx context.Context) *transportContractContext) func(*testing.T) {
	return func(t *testing.T) {
		test.WithContext(func(ctx context.Context) {
			c := makeContext(ctx)

			data := &TransportData{
				SenderPublicKey: c.publicKeys[3],
				RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
				Payloads:        [][]byte{{0x71, 0x72, 0x73}},
			}

			c.listeners[0].ExpectReceive(data.Payloads)
			c.listeners[1].ExpectReceive(data.Payloads)
			c.listeners[2].ExpectReceive(data.Payloads)
			c.listeners[3].ExpectNotReceive()

			c.transports[3].Send(ctx, data)
			c.verify(t)
		})
	}
}

type transportContractContext struct {
	publicKeys []primitives.Ed25519PublicKey
	transports []Transport
	listeners  []*MockTransportListener
}

func aChannelTransport(ctx context.Context) *transportContractContext {
	res := &transportContractContext{}
	res.publicKeys = []primitives.Ed25519PublicKey{{0x01}, {0x02}, {0x03}, {0x04}}

	federationNodes := make(map[string]config.FederationNode)
	for _, key := range res.publicKeys {
		federationNodes[key.KeyForMap()] = config.NewHardCodedFederationNode(primitives.Ed25519PublicKey(key))
	}

	logger := log.GetLogger(log.String("adapter", "transport"))

	transport := NewMemoryTransport(ctx, logger, federationNodes)
	res.transports = []Transport{transport, transport, transport, transport}
	res.listeners = []*MockTransportListener{
		listenTo(res.transports[0], res.publicKeys[0]),
		listenTo(res.transports[1], res.publicKeys[1]),
		listenTo(res.transports[2], res.publicKeys[2]),
		listenTo(res.transports[3], res.publicKeys[3]),
	}

	return res
}

func aDirectTransport(ctx context.Context) *transportContractContext {
	res := &transportContractContext{}

	firstRandomPort := test.RandomPort()
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < 4; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(firstRandomPort+i, "127.0.0.1")
		res.publicKeys = append(res.publicKeys, publicKey)
	}

	configs := []config.GossipTransportConfig{
		config.ForGossipAdapterTests(res.publicKeys[0], firstRandomPort+0, gossipPeers),
		config.ForGossipAdapterTests(res.publicKeys[1], firstRandomPort+1, gossipPeers),
		config.ForGossipAdapterTests(res.publicKeys[2], firstRandomPort+2, gossipPeers),
		config.ForGossipAdapterTests(res.publicKeys[3], firstRandomPort+3, gossipPeers),
	}

	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))

	res.transports = []Transport{
		NewDirectTransport(ctx, configs[0], logger),
		NewDirectTransport(ctx, configs[1], logger),
		NewDirectTransport(ctx, configs[2], logger),
		NewDirectTransport(ctx, configs[3], logger),
	}
	res.listeners = []*MockTransportListener{
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
