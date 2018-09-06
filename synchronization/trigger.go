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
	d       time.Duration
	f       func()
	ticker  *time.Ticker
	metrics *Telemetry
	running bool
	stop    chan bool
}

func NewPeriodicalTimer(interval time.Duration, trigger func()) PeriodicalTrigger {
	t := &periodicalTrigger{
		ticker:  time.NewTicker(interval),
		d:       interval,
		f:       trigger,
		metrics: &Telemetry{},
		stop:    make(chan bool),
		running: false,
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

func (t *periodicalTrigger) Start() {
	if t.running {
		return
	}
	t.running = true
	go func() {
		for {
			select {
			case <-t.ticker.C:
				t.f()
				t.metrics.timesTriggered++
			case <-t.stop:
				t.ticker.Stop()
				return
			}
		}
	}()
}

func (t *periodicalTrigger) FireNow() {
	t.Reset(t.d)
	go t.f()
	t.metrics.timesTriggeredManually++
}

func (t *periodicalTrigger) Reset(duration time.Duration) {
	t.Stop()
	t.metrics.timesReset++
	t.d = duration
	t.ticker = time.NewTicker(t.d)
	t.Start()
}

func (t *periodicalTrigger) Stop() {
	if !t.running {
		return
	}
	t.stop <- true
	t.running = false
}
