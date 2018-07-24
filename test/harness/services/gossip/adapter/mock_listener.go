package adapter

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type mockListener struct {
	mock.Mock
}

func (m *mockListener) OnTransportMessageReceived(payloads [][]byte) {
	m.Called(payloads)
}

func listenTo(transport adapter.Transport, publicKey primitives.Ed25519Pkey) *mockListener {
	l := &mockListener{}
	transport.RegisterListener(l, publicKey)
	return l
}

func (m *mockListener) expectReceive(payloads [][]byte) {
	m.WhenOnTransportMessageReceived(payloads).Return().Times(1)
}

func (m *mockListener) expectNotReceive() {
	m.WhenOnTransportMessageReceived(mock.Any).Return().Times(0)
}

func (m *mockListener) WhenOnTransportMessageReceived(arg interface{}) *mock.MockFunction {
	return m.When("OnTransportMessageReceived", arg)
}
