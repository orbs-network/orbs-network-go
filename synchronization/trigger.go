// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package synchronization

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"sync/atomic"
	"time"
)

type Telemetry struct {
	timesTriggered uint64
}

// the trigger is coupled with supervized package, this feels okay for now
type PeriodicalTrigger struct {
	govnr.TreeSupervisor
	d       time.Duration
	f       func()
	s       func()
	logger  logfields.Errorer
	cancel  context.CancelFunc
	ticker  *time.Ticker
	metrics *Telemetry
	Closed  govnr.ContextEndedChan
	name    string
}

func NewPeriodicalTrigger(ctx context.Context, name string, interval time.Duration, logger logfields.Errorer, trigger func(), onStop func()) *PeriodicalTrigger {
	subCtx, cancel := context.WithCancel(ctx)
	t := &PeriodicalTrigger{
		ticker:  nil,
		d:       interval,
		f:       trigger,
		s:       onStop,
		cancel:  cancel,
		logger:  logger,
		name:    name,
		metrics: &Telemetry{},
	}

	t.run(subCtx)
	return t
}

func (t *PeriodicalTrigger) TimesTriggered() uint64 {
	return atomic.LoadUint64(&t.metrics.timesTriggered)
}

func (t *PeriodicalTrigger) run(ctx context.Context) {
	t.ticker = time.NewTicker(t.d)
	h := govnr.Forever(ctx, t.name, logfields.GovnrErrorer(t.logger), func() {
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
	t.Closed = h.Done()
	t.Supervise(h)
}

func (t *PeriodicalTrigger) Stop() {
	t.cancel()
	// we want ticker stop to process before we return
	<-t.Closed
}
