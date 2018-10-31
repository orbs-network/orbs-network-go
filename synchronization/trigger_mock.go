package synchronization

import (
	"github.com/orbs-network/go-mock"
)

type PeriodicalTriggerMock struct {
	mock.Mock
}

func (m *PeriodicalTriggerMock) Stop() {
	m.Called()
}

func (m *PeriodicalTriggerMock) TimesTriggered() uint64 {
	m.Called()
	return 0
}

func (m *PeriodicalTriggerMock) IsRunning() bool {
	m.Called()
	return false
}
