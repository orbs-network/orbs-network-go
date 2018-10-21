package adapter

import (
	"context"
	"github.com/orbs-network/go-mock"
)

type transportListenerMock struct {
	mock.Mock
}

func (m *transportListenerMock) OnTransportMessageReceived(ctx context.Context, payloads [][]byte) {
	m.Called(ctx, payloads)
}
