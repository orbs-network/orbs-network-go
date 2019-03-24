// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
		electionChannel: make(chan func(ctx context.Context), 1), // buffered so trigger goroutine can terminate regardless of channel reader
		minTimeout:      minTimeout,
		onElectionCB:    onElectionCB,
		logger:          logger,
	}
}

func (e *exponentialBackoffElectionTrigger) RegisterOnElection(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, electionHandler func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics))) {
	if e.electionHandler == nil || e.view != view || e.blockHeight != blockHeight {
		timeout := e.CalcTimeout(view)
		e.view = view
		e.blockHeight = blockHeight
		e.Stop()
		e.triggerTimer = time.AfterFunc(timeout, e.sendTrigger)
		if e.view > 2 {
			e.logger.Info("Started election trigger", log.Uint64("block-height", uint64(e.blockHeight)), log.Uint64("lh-view", uint64(e.view)), log.String("lh-election-timeout", timeout.String()))
		}
	}
	e.electionHandler = electionHandler
}

func (e *exponentialBackoffElectionTrigger) ElectionChannel() chan func(ctx context.Context) {
	return e.electionChannel
}

func (e *exponentialBackoffElectionTrigger) Stop() {
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

func (e *exponentialBackoffElectionTrigger) sendTrigger() {
	e.logger.Info("election trigger triggered",
		log.Uint64("lh-election-block-height", uint64(e.blockHeight)),
		log.Uint64("lh-election-view", uint64(e.view)))
	e.electionChannel <- e.trigger
}

func (e *exponentialBackoffElectionTrigger) CalcTimeout(view primitives.View) time.Duration {
	timeoutMultiplier := time.Duration(int64(math.Pow(TIMEOUT_EXP_BASE, float64(view))))
	return timeoutMultiplier * e.minTimeout
}
