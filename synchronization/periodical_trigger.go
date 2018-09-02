package synchronization

import "time"

type PeriodicalTrigger interface {
	Reset()
	Cancel()
}

type Telemery struct {
	timesReset, timesTriggered int
}

// empty implementation - WIP
func NewPeriodicalTrigger(interval time.Duration, trigger func()) PeriodicalTrigger {
	return nil
}
