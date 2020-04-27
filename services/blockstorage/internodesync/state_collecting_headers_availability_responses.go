// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"time"
)

type collectingHeadersAvailabilityResponsesState struct {
	factory     *headerStateFactory
	client      *headerSyncClient
	createTimer func() *synchronization.Timer
	logger      log.Logger
	conduit     headerSyncConduit
	metrics     collectingStateMetrics
}

func (s *collectingHeadersAvailabilityResponsesState) name() string {
	return "collecting-headers-availability-responses"
}

func (s *collectingHeadersAvailabilityResponsesState) String() string {
	return s.name()
}

func (s *collectingHeadersAvailabilityResponsesState) processState(ctx context.Context) headerSyncState {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	start := time.Now()
	defer s.metrics.timeSpentInState.RecordSince(start) // runtime metric

	var responses []*gossipmessages.HeaderAvailabilityResponseMessage

	err := s.client.petitionerBroadcastHeaderAvailabilityRequest(ctx)
	if err != nil {
		logger.Info("failed to broadcast headers availability request", log.Error(err))
		s.metrics.timesFailedSendingAvailabilityRequest.Inc()
		return s.factory.CreateIdleState()
	}
	s.metrics.timesSucceededSendingAvailabilityRequest.Inc()
	waitForResponses := s.createTimer()
	for {
		select {
		case <-waitForResponses.C:
			logger.Info("finished waiting for responses", log.Int("responses-received", len(responses)))
			return s.factory.CreateFinishedCARState(responses)
		case e := <-s.conduit:
			switch r := e.(type) {
			case *gossipmessages.HeaderAvailabilityResponseMessage:
				responses = append(responses, r)
				logger.Info("got a new availability response", log.Stringable("response-source", r.Sender.SenderNodeAddress()), log.Stringable("first-block", r.SignedBatchRange.FirstBlockHeight()), log.Stringable("last-block", r.SignedBatchRange.LastBlockHeight()), log.Stringable("last-committed-block", r.SignedBatchRange.LastCommittedBlockHeight()))
			}
		case <-ctx.Done():
			return nil
		}
	}
}
