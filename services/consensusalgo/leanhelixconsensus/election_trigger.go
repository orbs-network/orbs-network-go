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
	clearTimer      chan bool
	logger          log.BasicLogger
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
	e.electionHandler = electionHandler
	if e.firstTime || e.view != view || e.blockHeight != blockHeight {
		e.firstTime = false
		e.view = view
		e.blockHeight = blockHeight
		e.stop(ctx)
		e.clearTimer = setTimeout(ctx, e.logger, e.onTimeout, e.CalcTimeout(view))
	}
}

func (e *exponentialBackoffElectionTrigger) ElectionChannel() chan func(ctx context.Context) {
	return e.electionChannel
}

func (e *exponentialBackoffElectionTrigger) stop(ctx context.Context) {
	if e.clearTimer != nil {
		select {
		case <-ctx.Done():
			return
		case e.clearTimer <- true:
			e.clearTimer = nil
		}
	}
}

func (e *exponentialBackoffElectionTrigger) trigger(ctx context.Context) {
	if e.electionHandler != nil {
		e.electionHandler(ctx, e.blockHeight, e.view, e.onElectionCB)
	}
}

func (e *exponentialBackoffElectionTrigger) onTimeout(ctx context.Context) {
	e.clearTimer = nil
	select {
	case <-ctx.Done():
		return
	case e.electionChannel <- e.trigger:
	}
}

func setTimeout(ctx context.Context, logger log.BasicLogger, cb func(ctx context.Context), timeout time.Duration) chan bool {
	timer := time.NewTimer(timeout)
	clear := make(chan bool)

	supervised.GoForever(ctx, logger, func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				cb(ctx)
				return
			case <-clear:
				timer.Stop()
				return
			}

		}
	})

	return clear
}

func (e *exponentialBackoffElectionTrigger) CalcTimeout(view primitives.View) time.Duration {
	timeoutMultiplier := time.Duration(int64(math.Pow(TIMEOUT_EXP_BASE, float64(view))))
	return timeoutMultiplier * e.minTimeout
}
