// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"time"
)

type idleResetMessage struct{}

type idleState struct {
	createTimer              func() *synchronization.Timer
	logger                   log.Logger
	factory                  *stateFactory
	conduit                  blockSyncConduit
	metrics                  idleStateMetrics
	management               services.Management
	managementReferenceGrace time.Duration
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
			if s.checkManagementReferenceIsUpToDate(ctx) {
				logger.Info("starting sync after no-commit timer expired")
				s.metrics.timesExpired.Inc()
				return s.factory.CreateCollectingAvailabilityResponseState()
			} else {
				return s.factory.CreateIdleState()
			}

		case <-ctx.Done():
			return nil
		}
	}
}


func (s *idleState) checkManagementReferenceIsUpToDate(ctx context.Context) bool {
	ref, err := s.management.GetCurrentReference(ctx, &services.GetCurrentReferenceInput{})
	if err != nil {
		s.logger.Error("management.GetCurrentReference should not return error", log.Error(err))
		return false
	}
	currentTime := primitives.TimestampSeconds(time.Now().Unix())
	managementGrace := primitives.TimestampSeconds(s.managementReferenceGrace / time.Second)
	if ref.CurrentReference + managementGrace < currentTime {
		s.logger.Error(fmt.Sprintf("management.GetCurrentReference(%d) is outdated compared to current time (%d) and allowed grace (%d)", ref.CurrentReference, currentTime, managementGrace))
		return false
	}
	return true
}