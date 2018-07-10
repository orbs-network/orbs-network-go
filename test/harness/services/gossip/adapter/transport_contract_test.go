package adapter

import (
	"testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/orbs-network/orbs-network-go/test"
	"github.com/maraino/go-mock"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
)

func TestContract(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gossip Transport Contract")
}

var _ = Describe("Tempering Transport", func() {
	assertContractOf(aTemperingTransport)
})

//var _ = Describe("TCP Transport", func() {
//	assertContractOf(aTCPTransport())
//})

func assertContractOf(makeContext func() transportContractContext) {
	When("unicasting a message", func() {

		It("reaches only the intended recipient", func() {
			c := makeContext()
			message := &adapter.Message{}
			c.l2.expect(message)
			c.transport.Unicast("l2", message)
			c.verify()
		})
	})

	When("broadcasting a message", func() {
		It("reaches all recipients", func() {
			c := makeContext()
			message := &adapter.Message{}
			c.l1.expect(message)
			c.l2.expect(message)
			c.l3.expect(message)
			c.transport.Broadcast(message)
			c.verify()
		})
	})
}

type mockListener struct {
	mock.Mock
}

func (l *mockListener) OnTransportMessageReceived(message *adapter.Message) {
	l.Called(message)
}

func listenTo(transport adapter.Transport, name string) *mockListener {
	l := &mockListener{}
	transport.RegisterListener(l, name)
	return l
}

func (l *mockListener) expect(m *adapter.Message) {
	l.When("OnMessageReceived", m).Return().Times(1)
}

type transportContractContext struct {
	l1, l2, l3 *mockListener
	transport  adapter.Transport
}

func aTemperingTransport() transportContractContext {
	transport := NewTemperingTransport()
	l1 := listenTo(transport, "l1")
	l2 := listenTo(transport, "l2")
	l3 := listenTo(transport, "l3")
	return transportContractContext{l1, l2, l3, transport}
}

func (c transportContractContext) verify() {
	Eventually(c.l1).Should(ExecuteAsPlanned())
	Eventually(c.l2).Should(ExecuteAsPlanned())
	Eventually(c.l3).Should(ExecuteAsPlanned())
}
