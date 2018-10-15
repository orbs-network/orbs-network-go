package synchronization

import (
	"context"
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
	d       time.Duration
	f       func()
	ticker  *time.Ticker
	metrics *Telemetry
	running bool
	stop    chan struct{}
	wgSync  sync.WaitGroup
	ctx     context.Context
}

func NewPeriodicalTrigger(ctx context.Context, interval time.Duration, trigger func()) Trigger {
	t := &periodicalTrigger{
		ticker:  nil,
		d:       interval,
		f:       trigger,
		metrics: &Telemetry{},
		stop:    make(chan struct{}),
		running: false,
		ctx:     ctx,
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
			case <-t.ctx.Done():
				t.ticker.Stop()
				t.running = false
				t.wgSync.Done()
				return
			}
		}
	}()
}

func (t *periodicalTrigger) FireNow() {
	t.reset(t.d, true)
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

	t.stop <- struct{}{}
	// we set running to false only once the gofunc terminates and we will block (possibly, for a few nanosecs) until that channel is processed
	t.wgSync.Wait()
}
