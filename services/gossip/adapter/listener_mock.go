package adapter

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type MockTransportListener struct {
	mock.Mock
}

func (l *MockTransportListener) OnTransportMessageReceived(ctx context.Context, payloads [][]byte) {
	l.Called(ctx, payloads)
}

func listenTo(transport Transport, publicKey primitives.Ed25519PublicKey) *MockTransportListener {
	l := &MockTransportListener{}
	transport.RegisterListener(l, publicKey)
	return l
}

func (l *MockTransportListener) ExpectReceive(payloads [][]byte) {
	l.WhenOnTransportMessageReceived(payloads).Return().Times(1)
}

func (l *MockTransportListener) ExpectNotReceive() {
	l.Never("OnTransportMessageReceived", mock.Any, mock.Any)
}

func (l *MockTransportListener) WhenOnTransportMessageReceived(arg interface{}) *mock.MockFunction {
	return l.When("OnTransportMessageReceived", mock.Any, arg)
}
