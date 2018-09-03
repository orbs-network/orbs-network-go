package synchronization

import "time"

type PeriodicalTrigger interface {
	Reset()
	Cancel()
}

type Telemetry struct {
	timesReset, timesTriggered int
}

type periodicalTrigger struct {
	timer    *time.Timer
	interval time.Duration
}

// empty implementation - WIP
func NewPeriodicalTrigger(interval time.Duration, trigger func()) PeriodicalTrigger {
	t := &periodicalTrigger{
		timer:    time.AfterFunc(interval, trigger),
		interval: interval,
	}
	return t
}

func (t *periodicalTrigger) Reset() {
	t.timer.Stop()
	t.timer.Reset(t.interval)
}

func (t *periodicalTrigger) Cancel() {
	t.timer.Stop()
}
