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
	t.Run("MemberlistTransport", broadcastTest(aMemberlistTransport))
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

			c.transports[3].Send(data)
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

func aMemberlistTransport(ctx context.Context) *transportContractContext {
	res := &transportContractContext{}
	res.publicKeys = []primitives.Ed25519PublicKey{{0x01}, {0x02}, {0x03}, {0x04}}
	configs := []adapter.MemberlistGossipConfig{
		{res.publicKeys[0], 60001, []string{"127.0.0.1:60002", "127.0.0.1:60003", "127.0.0.1:60004"}},
		{res.publicKeys[1], 60002, []string{"127.0.0.1:60001", "127.0.0.1:60003", "127.0.0.1:60004"}},
		{res.publicKeys[2], 60003, []string{"127.0.0.1:60001", "127.0.0.1:60002", "127.0.0.1:60004"}},
		{res.publicKeys[3], 60004, []string{"127.0.0.1:60001", "127.0.0.1:60002", "127.0.0.1:60003"}},
	}
	res.transports = []adapter.Transport{
		adapter.NewMemberlistTransport(configs[0]),
		adapter.NewMemberlistTransport(configs[1]),
		adapter.NewMemberlistTransport(configs[2]),
		adapter.NewMemberlistTransport(configs[3]),
	}
	res.listeners = []*mockListener{
		listenTo(res.transports[0], res.publicKeys[0]),
		listenTo(res.transports[1], res.publicKeys[1]),
		listenTo(res.transports[2], res.publicKeys[2]),
		listenTo(res.transports[3], res.publicKeys[3]),
	}
	return res
}

func createContractTestConfig(publicKey primitives.Ed25519PublicKey, federationNodes map[string]config.FederationNode) adapter.Config {
	cfg := config.EmptyConfig()
	cfg.SetNodePublicKey(publicKey)
	cfg.SetFederationNodes(federationNodes)
	cfg.SetDuration(config.GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(config.GOSSIP_NETWORK_TIMEOUT, 1*time.Second)
	return cfg
}

func aDirectTransport(ctx context.Context) *transportContractContext {
	res := &transportContractContext{}

	// randomize listen port between tests to reduce flakiness and chances of listening clashes
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	firstRandomPort := 20000 + r.Intn(40000)

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < 4; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey, uint16(firstRandomPort+i), "127.0.0.1")
		res.publicKeys = append(res.publicKeys, publicKey)
	}

	configs := []adapter.Config{
		createContractTestConfig(res.publicKeys[0], federationNodes),
		createContractTestConfig(res.publicKeys[1], federationNodes),
		createContractTestConfig(res.publicKeys[2], federationNodes),
		createContractTestConfig(res.publicKeys[3], federationNodes),
	}

	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	res.transports = []adapter.Transport{
		adapter.NewDirectTransport(ctx, configs[0], log),
		adapter.NewDirectTransport(ctx, configs[1], log),
		adapter.NewDirectTransport(ctx, configs[2], log),
		adapter.NewDirectTransport(ctx, configs[3], log),
	}
	res.listeners = []*mockListener{
		listenTo(res.transports[0], res.publicKeys[0]),
		listenTo(res.transports[1], res.publicKeys[1]),
		listenTo(res.transports[2], res.publicKeys[2]),
		listenTo(res.transports[3], res.publicKeys[3]),
	}

	// TODO: improve this, we need some time until everybody connects to everybody else
	// TODO: maybe add an adapter function to check how many active outgoing connections we have
	time.Sleep(2 * configs[0].GossipConnectionKeepAliveInterval())

	return res
}

func (c *transportContractContext) verify(t *testing.T) {
	for _, mockListener := range c.listeners {
		require.NoError(t, test.EventuallyVerify(mockListener))
	}
}
