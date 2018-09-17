package synchronization

import (
	"sync/atomic"
	"time"
)

type Trigger interface {
	Start()
	Reset(duration time.Duration)
	TimesTriggered() uint64
	TimesReset() uint64
	TimesTriggeredManually() uint64
	IsRunning() bool
	FireNow()
	Stop()
}

type Telemetry struct {
	timesReset, timesTriggered, timesTriggeredManually uint64
}

type periodicalTrigger struct {
	d          time.Duration
	f          func()
	ticker     *time.Ticker
	timer      *time.Timer
	metrics    *Telemetry
	running    bool
	stop       chan struct{}
	periodical bool
}

func NewTrigger(interval time.Duration, trigger func()) Trigger {
	t := &periodicalTrigger{
		ticker:     nil,
		timer:      nil,
		d:          interval,
		f:          trigger,
		metrics:    &Telemetry{},
		stop:       nil,
		running:    false,
		periodical: false,
	}
	return t
}

func NewPeriodicalTrigger(interval time.Duration, trigger func()) Trigger {
	t := &periodicalTrigger{
		ticker:     nil,
		timer:      nil,
		d:          interval,
		f:          trigger,
		metrics:    &Telemetry{},
		stop:       make(chan struct{}),
		running:    false,
		periodical: true,
	}
	return t
}

func (t *periodicalTrigger) IsRunning() bool {
	return t.running
}

func (t *periodicalTrigger) TimesTriggered() uint64 {
	return atomic.LoadUint64(&t.metrics.timesTriggered)
}

func (t *periodicalTrigger) TimesReset() uint64 {
	return atomic.LoadUint64(&t.metrics.timesReset)
}

func (t *periodicalTrigger) TimesTriggeredManually() uint64 {
	return atomic.LoadUint64(&t.metrics.timesTriggeredManually)
}

func (t *periodicalTrigger) Start() {
	if t.running {
		return
	}
	t.running = true
	if t.periodical {
		t.ticker = time.NewTicker(t.d)
		go func() {
			for {
				select {
				case <-t.ticker.C:
					t.f()
					atomic.AddUint64(&t.metrics.timesTriggered, 1)
				case <-t.stop:
					t.ticker.Stop()
					return
				}
			}
		}()
	} else {
		t.timer = time.AfterFunc(t.d, t.f)
	}
}

func (t *periodicalTrigger) FireNow() {
	if !t.periodical {
		t.Stop()
	} else {
		t.reset(t.d, true)
	}
	go t.f()
	atomic.AddUint64(&t.metrics.timesTriggeredManually, 1)
}

func (t *periodicalTrigger) Reset(duration time.Duration) {
	t.reset(duration, false)
}

func (t *periodicalTrigger) reset(duration time.Duration, internal bool) {
	t.Stop()
	if !internal {
		atomic.AddUint64(&t.metrics.timesReset, 1)
	}
	t.d = duration
	t.Start()
}

func (t *periodicalTrigger) Stop() {
	if !t.running {
		return
	}

	if t.periodical {
		t.stop <- struct{}{}
	} else {
		t.timer.Stop()
	}

	t.running = false
}
