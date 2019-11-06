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
	"time"
)

// proxy interface for time.Ticker
type Ticker interface {
	// The channel on which the ticks are delivered.
	C() <-chan time.Time
	// Stop turns off a ticker. After Stop, no more ticks will be sent.
	// Stop does not close the channel, to prevent a concurrent goroutine
	// reading from the channel from seeing an erroneous "tick".
	Stop()
}

type timeTicker struct {
	time.Ticker
}

func NewTimeTicker(d time.Duration) Ticker {
	return &timeTicker{*time.NewTicker(d)}
}

func (t *timeTicker) C() <-chan time.Time {
	return t.Ticker.C
}

type HookTicker struct {
	C_    func() <-chan time.Time
	Stop_ func()
}

func (t *HookTicker) C() <-chan time.Time {
	return t.C_()
}

func (t *HookTicker) Stop() {
	t.Stop_()
}

var noopChannel = make(chan time.Time)

func NewHookTicker() *HookTicker {
	return &HookTicker{
		C_:    func() <-chan time.Time { return noopChannel },
		Stop_: func() {},
	}
}

// the trigger is coupled with supervized package, this feels okay for now
type PeriodicalTrigger struct {
	govnr.TreeSupervisor
	handler func()
	onStop  func()
	logger  logfields.Errorer
	cancel  context.CancelFunc
	ticker  Ticker
	Closed  govnr.ContextEndedChan
	name    string
}

func NewPeriodicalTrigger(ctx context.Context, name string, ticker Ticker, logger logfields.Errorer, trigger func(), onStop func()) *PeriodicalTrigger {
	subCtx, cancel := context.WithCancel(ctx)
	t := &PeriodicalTrigger{
		ticker:  ticker,
		handler: trigger,
		onStop:  onStop,
		cancel:  cancel,
		logger:  logger,
		name:    name,
	}

	t.run(subCtx)
	return t
}

func (t *PeriodicalTrigger) run(ctx context.Context) {
	h := govnr.Forever(ctx, t.name, logfields.GovnrErrorer(t.logger), func() {
		for {
			select {
			case <-t.ticker.C():
				t.handler()
			case <-ctx.Done():
				t.ticker.Stop()
				if t.onStop != nil {
					go t.onStop()
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
