package leanhelixconsensus

import (
	"context"
	lhmetrics "github.com/orbs-network/lean-helix-go/instrumentation/metrics"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	"github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"math"
	"time"
)

var TIMEOUT_EXP_BASE = float64(2.0) // Modifying this value from 2.0 will affect its unit tests which are time-based

type exponentialBackoffElectionTrigger struct {
	electionChannel chan func(ctx context.Context)
	minTimeout      time.Duration
	view            primitives.View
	blockHeight     primitives.BlockHeight
	electionHandler func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics))
	onElectionCB    func(m lhmetrics.ElectionMetrics)
	logger          log.BasicLogger
	triggerTimer    *time.Timer
}

func NewExponentialBackoffElectionTrigger(logger log.BasicLogger, minTimeout time.Duration, onElectionCB func(m lhmetrics.ElectionMetrics)) lh.ElectionTrigger {

	return &exponentialBackoffElectionTrigger{
		electionChannel: make(chan func(ctx context.Context)),
		minTimeout:      minTimeout,
		onElectionCB:    onElectionCB,
		logger:          logger,
	}
}

func (e *exponentialBackoffElectionTrigger) RegisterOnElection(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, electionHandler func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics))) {
	e.logger.Info("ElectionTrigger registration start")
	if e.electionHandler == nil || e.view != view || e.blockHeight != blockHeight {
		timeout := e.CalcTimeout(view)
		e.view = view
		e.blockHeight = blockHeight
		e.safeTimerStop()
		wrappedTrigger := func() { e.sendTrigger(ctx) }
		e.triggerTimer = time.AfterFunc(timeout, wrappedTrigger)
		e.logger.Info("ElectionTrigger restarted timer for height and view",
			log.Uint64("lh-election-block-height", uint64(e.blockHeight)),
			log.Uint64("lh-election-view", uint64(e.view)),
			log.Stringable("lh-election-timeout", timeout))

	}
	e.electionHandler = electionHandler
}

func (e *exponentialBackoffElectionTrigger) ElectionChannel() chan func(ctx context.Context) {
	return e.electionChannel
}

func (e *exponentialBackoffElectionTrigger) safeTimerStop() {
	if e.triggerTimer != nil {
		active := e.triggerTimer.Stop()
		if !active {
			select {
			case <-e.triggerTimer.C:
			default:
			}
		}
	}
}

func (e *exponentialBackoffElectionTrigger) trigger(ctx context.Context) {
	if e.electionHandler != nil {
		e.electionHandler(ctx, e.blockHeight, e.view, e.onElectionCB)
	}
}

func (e *exponentialBackoffElectionTrigger) sendTrigger(ctx context.Context) {
	e.logger.Info("ElectionTrigger triggered timeout")
	select {
	case <-ctx.Done():
		return
	case e.electionChannel <- e.trigger:
	}
}

func (e *exponentialBackoffElectionTrigger) CalcTimeout(view primitives.View) time.Duration {
	timeoutMultiplier := time.Duration(int64(math.Pow(TIMEOUT_EXP_BASE, float64(view))))
	return timeoutMultiplier * e.minTimeout
}
