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

func (m *PeriodicalTriggerMock) Cancel() {
	m.Called()
}
