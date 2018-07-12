package adapter

import (
	"testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orbs-network/orbs-network-go/test"
	"github.com/maraino/go-mock"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

func TestContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gossip Transport Contract")
}

var _ = Describe("Tempering Transport", func() {
	assertContractOf(aTemperingTransport)
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
		It("reaches all recipients", func() {
			c := makeContext()
			header := (&gossipmessages.HeaderBuilder{
				RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
				Topic: gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY,
				TransactionRelay: gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS,
				NumPayloads: 0,
			}).Build()
			payloads := [][]byte{}
			c.l1.expect(header, payloads)
			c.l2.expect(header, payloads)
			c.l3.expect(header, payloads)
			c.transport.Send(header, payloads)
			c.verify()
		})
	})
}

type mockListener struct {
	mock.Mock
}

func (m *mockListener) OnTransportMessageReceived(header *gossipmessages.Header, payloads [][]byte) {
	m.Called(header, payloads)
}

func listenTo(transport adapter.Transport, name string) *mockListener {
	l := &mockListener{}
	transport.RegisterListener(l, name)
	return l
}

func (m *mockListener) expect(header *gossipmessages.Header, payloads [][]byte) {
	m.When("OnTransportMessageReceived", header, payloads).Return().Times(1)
}

type transportContractContext struct {
	l1, l2, l3 *mockListener
	transport  adapter.Transport
}

func aTemperingTransport() *transportContractContext {
	transport := NewTemperingTransport()
	l1 := listenTo(transport, "l1")
	l2 := listenTo(transport, "l2")
	l3 := listenTo(transport, "l3")
	return &transportContractContext{l1, l2, l3, transport}
}

func (c *transportContractContext) verify() {
	Eventually(c.l1).Should(ExecuteAsPlanned())
	Eventually(c.l2).Should(ExecuteAsPlanned())
	Eventually(c.l3).Should(ExecuteAsPlanned())
}