package synchronization

import (
	"sync"
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
	wgSync     sync.WaitGroup
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
		t.wgSync.Add(1)
		go func() {
			for {
				select {
				case <-t.ticker.C:
					t.f()
					atomic.AddUint64(&t.metrics.timesTriggered, 1)
				case <-t.stop:
					t.ticker.Stop()
					t.running = false
					t.wgSync.Done()
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
	// its possible in a periodical mode, that stop did not happen by this time, we do not want to start until it does
	t.wgSync.Wait()
	t.Start()
}

func (t *periodicalTrigger) Stop() {
	if !t.running {
		return
	}

	if t.periodical {
		t.stop <- struct{}{}
		// we set running to false only once the gofunc terminates (other side of this channel)
	} else {
		t.timer.Stop()
		t.running = false
	}
}
