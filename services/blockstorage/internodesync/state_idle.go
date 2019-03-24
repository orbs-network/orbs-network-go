// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"time"
)

type idleResetMessage struct{}

type idleState struct {
	createTimer func() *synchronization.Timer
	logger      log.BasicLogger
	factory     *stateFactory
	conduit     blockSyncConduit
	metrics     idleStateMetrics
}

func (s *idleState) name() string {
	return "idle-state"
}

func (s *idleState) String() string {
	return s.name()
}

func (s *idleState) processState(ctx context.Context) syncState {
	start := time.Now()
	defer s.metrics.timeSpentInState.RecordSince(start) // runtime metric
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	noCommitTimer := s.createTimer()
	for {
		select {
		case e := <-s.conduit:
			switch e.(type) {
			case idleResetMessage:
				s.metrics.timesReset.Inc()
				return s.factory.CreateIdleState()
			}
		case <-noCommitTimer.C:
			logger.Info("starting sync after no-commit timer expired")
			s.metrics.timesExpired.Inc()
			return s.factory.CreateCollectingAvailabilityResponseState()
		case <-ctx.Done():
			return nil
		}
	}
}
