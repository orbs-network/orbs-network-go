package leanhelixconsensus

import (
	"context"
	lhmetrics "github.com/orbs-network/lean-helix-go/instrumentation/metrics"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	"github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"math"
	"time"
)

var TIMEOUT_EXP_BASE = float64(2.0) // Modifying this value from 2.0 will affect its unit tests which are time-based

type exponentialBackoffElectionTrigger struct {
	electionChannel chan func(ctx context.Context)
	minTimeout      time.Duration
	view            primitives.View
	blockHeight     primitives.BlockHeight
	firstTime       bool
	electionHandler func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics))
	onElectionCB    func(m lhmetrics.ElectionMetrics)
	logger          log.BasicLogger
	timer           *time.Timer
}

func NewExponentialBackoffElectionTrigger(logger log.BasicLogger, minTimeout time.Duration, onElectionCB func(m lhmetrics.ElectionMetrics)) lh.ElectionTrigger {

	return &exponentialBackoffElectionTrigger{
		electionChannel: make(chan func(ctx context.Context)),
		minTimeout:      minTimeout,
		firstTime:       true,
		onElectionCB:    onElectionCB,
		logger:          logger,
	}
}

func (e *exponentialBackoffElectionTrigger) RegisterOnElection(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, electionHandler func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics))) {
	e.logger.Info("RegisterOnElection() start")
	e.electionHandler = electionHandler
	if e.firstTime || e.view != view || e.blockHeight != blockHeight {
		e.firstTime = false
		e.view = view
		e.view = view
		e.blockHeight = blockHeight
		e.logger.Info("stop() successful, clearTimer is not nil")
		e.restartTimer(ctx, e.logger, e.onTimeout, e.CalcTimeout(view))
	}
}

func (e *exponentialBackoffElectionTrigger) ElectionChannel() chan func(ctx context.Context) {
	return e.electionChannel
}

func (e *exponentialBackoffElectionTrigger) tryStop() {
	if e.timer != nil {
		e.timer.Stop()
	}
}

func (e *exponentialBackoffElectionTrigger) trigger(ctx context.Context) {
	if e.electionHandler != nil {
		e.electionHandler(ctx, e.blockHeight, e.view, e.onElectionCB)
	}
}

func (e *exponentialBackoffElectionTrigger) onTimeout(ctx context.Context) {
	//e.clearTimer = nil
	e.logger.Info("onTimeout() start, clearTimer is nil")
	select {
	case <-ctx.Done():
		return
	case e.electionChannel <- e.trigger:
	}
}

func (e *exponentialBackoffElectionTrigger) restartTimer(ctx context.Context, logger log.BasicLogger, cb func(ctx context.Context), timeout time.Duration) {

	e.tryStop()
	e.timer = time.NewTimer(timeout)

	supervised.GoOnce(logger, func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-e.timer.C:
				cb(ctx)
				return
			}
		}
	})
}

func (e *exponentialBackoffElectionTrigger) CalcTimeout(view primitives.View) time.Duration {
	timeoutMultiplier := time.Duration(int64(math.Pow(TIMEOUT_EXP_BASE, float64(view))))
	return timeoutMultiplier * e.minTimeout
}
