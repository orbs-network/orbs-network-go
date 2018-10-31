package synchronization

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Trigger interface {
	TimesTriggered() uint64
	IsRunning() bool
	Stop()
}

type Telemetry struct {
	timesTriggered uint64
}

type periodicalTrigger struct {
	d       time.Duration
	f       func()
	s       func()
	ticker  *time.Ticker
	metrics *Telemetry
	running bool
	stop    chan struct{}
	wgSync  sync.WaitGroup
}

func NewPeriodicalTrigger(ctx context.Context, interval time.Duration, trigger func(), onStop func()) Trigger {
	t := &periodicalTrigger{
		ticker:  nil,
		d:       interval,
		f:       trigger,
		s:       onStop,
		metrics: &Telemetry{},
		stop:    make(chan struct{}),
		running: false,
	}
	t.run(ctx)
	return t
}

func (t *periodicalTrigger) IsRunning() bool {
	return t.running
}

func (t *periodicalTrigger) TimesTriggered() uint64 {
	return atomic.LoadUint64(&t.metrics.timesTriggered)
}

func (t *periodicalTrigger) run(ctx context.Context) {
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
				if t.s != nil {
					go t.s()
				}
				return
			case <-ctx.Done():
				t.ticker.Stop()
				t.running = false
				t.wgSync.Done()
				if t.s != nil {
					go t.s()
				}
				return
			}
		}
	}()
}

func (t *periodicalTrigger) Stop() {
	if !t.running {
		return
	}

	t.stop <- struct{}{}
	// we set running to false only once the gofunc terminates and we will block (possibly, for a few nanosecs) until that channel is processed
	t.wgSync.Wait()
}
