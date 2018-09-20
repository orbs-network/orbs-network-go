package adapter

import (
	"github.com/orbs-network/go-mock"
)

type transportListenerMock struct {
	mock.Mock
}

func (m *transportListenerMock) OnTransportMessageReceived(payloads [][]byte) {
	m.Called(payloads)
}
