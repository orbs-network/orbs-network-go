package adapter

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type MockTransportListener struct {
	mock.Mock
}

func (m *MockTransportListener) OnTransportMessageReceived(ctx context.Context, payloads [][]byte) {
	m.Called(ctx, payloads)
}

func listenTo(transport Transport, publicKey primitives.Ed25519PublicKey) *MockTransportListener {
	l := &MockTransportListener{}
	transport.RegisterListener(l, publicKey)
	return l
}

func (m *MockTransportListener) ExpectReceive(payloads [][]byte) {
	m.WhenOnTransportMessageReceived(payloads).Return().Times(1)
}

func (m *MockTransportListener) ExpectNotReceive() {
	m.Never("OnTransportMessageReceived", mock.Any, mock.Any)
}

func (m *MockTransportListener) WhenOnTransportMessageReceived(arg interface{}) *mock.MockFunction {
	return m.When("OnTransportMessageReceived", mock.Any, arg)
}
