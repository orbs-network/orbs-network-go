// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package synchronization

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"sync"
	"sync/atomic"
	"time"
)

type Trigger interface {
	TimesTriggered() uint64
	Stop()
}

type Telemetry struct {
	timesTriggered uint64
}

// the trigger is coupled with supervized package, this feels okay for now
type periodicalTrigger struct {
	d       time.Duration
	f       func()
	s       func()
	logger  supervised.Errorer
	cancel  context.CancelFunc
	ticker  *time.Ticker
	metrics *Telemetry
	wgSync  sync.WaitGroup
}

func NewPeriodicalTrigger(ctx context.Context, interval time.Duration, logger supervised.Errorer, trigger func(), onStop func()) Trigger {
	subCtx, cancel := context.WithCancel(ctx)
	t := &periodicalTrigger{
		ticker:  nil,
		d:       interval,
		f:       trigger,
		s:       onStop,
		cancel:  cancel,
		logger:  logger,
		metrics: &Telemetry{},
	}

	t.run(subCtx)
	return t
}

func (t *periodicalTrigger) TimesTriggered() uint64 {
	return atomic.LoadUint64(&t.metrics.timesTriggered)
}

func (t *periodicalTrigger) run(ctx context.Context) {
	t.ticker = time.NewTicker(t.d)
	supervised.GoForever(ctx, t.logger, func() {
		t.wgSync.Add(1)
		defer t.wgSync.Done()
		for {
			select {
			case <-t.ticker.C:
				t.f()
				atomic.AddUint64(&t.metrics.timesTriggered, 1)
			case <-ctx.Done():
				t.ticker.Stop()
				if t.s != nil {
					go t.s()
				}
				return
			}
		}
	})
}

func (t *periodicalTrigger) Stop() {
	t.cancel()
	// we want ticker stop to process before we return
	t.wgSync.Wait()
}
