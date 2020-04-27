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
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"math/rand"
	"time"
)

type finishedCHARState struct {
	responses []*gossipmessages.HeaderAvailabilityResponseMessage
	logger    log.Logger
	factory   *headerStateFactory
	metrics   finishedCollectingStateMetrics
}

func (s *finishedCHARState) name() string {
	return "finished-collecting-headers-availability-responses-state"
}

func (s *finishedCHARState) String() string {
	return fmt.Sprintf("%s-with-%d-responses", s.name(), len(s.responses))
}

func (s *finishedCHARState) processState(ctx context.Context) headerSyncState {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	start := time.Now()
	defer s.metrics.timeSpentInState.RecordSince(start) // runtime metric

	c := len(s.responses)
	if c == 0 {
		logger.Info("no responses received")
		s.metrics.finishedWithNoResponsesCount.Inc()
		return s.factory.CreateIdleState()
	}
	s.metrics.finishedWithSomeResponsesCount.Inc()
	randomSourceIdx := rand.Intn(len(s.responses))
	syncSource := s.responses[randomSourceIdx]
	logger.Info("selecting from sync sources", log.Int("sources-count", c), log.Int("selected", randomSourceIdx), log.String("selected-address", syncSource.Sender.StringSenderNodeAddress()))
	syncSourceNodeAddress := syncSource.Sender.SenderNodeAddress()

	if !s.factory.conduit.drainAndCheckForShutdown(ctx) {
		return nil
	}
	return s.factory.CreateWaitingForChunksState(syncSourceNodeAddress)
}
