package synchronization

import (
	"context"
	"github.com/orbs-network/go-mock"
	"time"
)

type PeriodicalTriggerMock struct {
	mock.Mock
}

func (m *PeriodicalTriggerMock) Reset(ctx context.Context, duration time.Duration) {
	m.Called()
}

func (m *PeriodicalTriggerMock) Stop() {
	m.Called()
}

func (m *PeriodicalTriggerMock) FireNow(ctx context.Context) {
	m.Called()
}

func (m *PeriodicalTriggerMock) Start(ctx context.Context) {
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
