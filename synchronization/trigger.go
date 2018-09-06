package synchronization

import "time"

type PeriodicalTrigger interface {
	Start()
	Reset(duration time.Duration)
	TimesTriggered() uint
	TimesReset() uint
	TimesTriggeredManually() uint
	IsRunning() bool
	FireNow()
	Stop()
}

type Telemetry struct {
	timesReset, timesTriggered, timesTriggeredManually uint
}

type periodicalTrigger struct {
	d          time.Duration
	f          func()
	ticker     *time.Timer
	metrics    *Telemetry
	running    bool
	periodical bool
}

func NewTrigger(interval time.Duration, trigger func(), periodical bool) PeriodicalTrigger {
	t := &periodicalTrigger{
		ticker:     nil,
		d:          interval,
		f:          trigger,
		metrics:    &Telemetry{},
		running:    false,
		periodical: periodical,
	}
	return t
}

func (t *periodicalTrigger) IsRunning() bool {
	return t.running
}

func (t *periodicalTrigger) TimesTriggered() uint {
	return t.metrics.timesTriggered
}

func (t *periodicalTrigger) TimesReset() uint {
	return t.metrics.timesReset
}

func (t *periodicalTrigger) TimesTriggeredManually() uint {
	return t.metrics.timesTriggeredManually
}

func (t *periodicalTrigger) getWrappedFunc() func() {
	return func() {
		if t.periodical {
			t.reset(t.d, true)
		}
		t.f()
		t.metrics.timesTriggered++
	}
}

func (t *periodicalTrigger) Start() {
	if t.running {
		return
	}
	t.running = true
	t.ticker = time.AfterFunc(t.d, t.getWrappedFunc())
}

func (t *periodicalTrigger) FireNow() {
	go t.f()
	t.Reset(t.d)
	t.metrics.timesTriggeredManually++
}

func (t *periodicalTrigger) Reset(duration time.Duration) {
	t.reset(duration, false)
}

func (t *periodicalTrigger) reset(duration time.Duration, internal bool) {
	t.Stop()
	t.d = duration
	t.Start()
	if !internal {
		t.metrics.timesReset++
	}
}

func (t *periodicalTrigger) Stop() {
	if !t.running {
		return
	}
	t.ticker.Stop()
	t.running = false
}
