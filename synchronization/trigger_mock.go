package synchronization

import (
	"github.com/orbs-network/go-mock"
	"time"
)

type PeriodicalTriggerMock struct {
	mock.Mock
}

func (m *PeriodicalTriggerMock) Reset(duration time.Duration) {
	m.Called()
}

func (m *PeriodicalTriggerMock) Stop() {
	m.Called()
}

func (m *PeriodicalTriggerMock) FireNow() {
	m.Called()
}

func (m *PeriodicalTriggerMock) Start() {
	m.Called()
}

func (m *PeriodicalTriggerMock) TimesTriggered() uint64 {
	m.Called()
	return 0
}

func (m *PeriodicalTriggerMock) TimesReset() uint64 {
	m.Called()
	return 0
}

func (m *PeriodicalTriggerMock) TimesTriggeredManually() uint64 {
	m.Called()
	return 0
}

func (m *PeriodicalTriggerMock) IsRunning() bool {
	m.Called()
	return false
}
