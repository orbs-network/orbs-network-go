package synchronization

import (
	"github.com/orbs-network/go-mock"
)

type PeriodicalTriggerMock struct {
	mock.Mock
}

func (m *PeriodicalTriggerMock) Reset() {
	m.Called()
}

func (m *PeriodicalTriggerMock) Cancel() {
	m.Called()
}
