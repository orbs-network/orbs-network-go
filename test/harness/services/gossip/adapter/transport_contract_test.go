package adapter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"testing"
)

func TestContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gossip Transport Contract")
}

var _ = Describe("Tampering Transport", func() {
	assertContractOf(aTamperingTransport)
})

var _ = Describe("Memberlist Transport", func() {
	assertContractOf(aMemberlistTransport)
})

func assertContractOf(makeContext func() *transportContractContext) {

	/* // TODO: add me
	When("unicasting a message", func() {

		It("reaches only the intended recipient", func() {
			c := makeContext()
			header := (&gossipmessages.HeaderBuilder{
				RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
				Topic: gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY,
				TransactionRelay: gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS,
				NumPayloads: 0,
			}).Build()
			payloads := [][]byte{}
			c.l2.expect(header, payloads)
			c.transport.Send(header, payloads)
			c.verify()
		})
	})
	*/

	When("broadcasting a message", func() {
		It("reaches all recipients except the sender", func() {
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
			c.verify()
		})
	})
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

func (c *transportContractContext) verify() {
	for _, mockListener := range c.listeners {
		Eventually(mockListener).Should(test.ExecuteAsPlanned())
	}
}
