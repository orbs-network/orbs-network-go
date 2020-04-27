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
	"time"
)

type processingHeadersState struct {
	headers *gossipmessages.HeaderSyncResponseMessage
	logger  log.Logger
	storage BlockSyncStorage
	factory *headerStateFactory
	metrics processingStateMetrics
}

func (s *processingHeadersState) name() string {
	return "processing-headers-state"
}

func (s *processingHeadersState) String() string {
	if s.headers != nil {
		return fmt.Sprintf("%s-with-%d-headers", s.name(), len(s.headers.HeaderWithProof))
	}

	return s.name()
}

func (s *processingHeadersState) processState(ctx context.Context) headerSyncState {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	start := time.Now()
	defer s.metrics.timeSpentInState.RecordSince(start) // runtime metric

	if s.headers == nil {
		s.logger.Info("possible byzantine state in header sync, received no headers to processing headers state")
		return s.factory.CreateIdleState()
	}

	firstBlockHeight := s.headers.SignedChunkRange.FirstBlockHeight()
	lastBlockHeight := s.headers.SignedChunkRange.LastBlockHeight()

	numHeaders := len(s.headers.HeaderWithProof)
	logger.Info("committing headers from sync",
		log.Int("headers-count", numHeaders),
		log.Stringable("sender", s.headers.Sender),
		log.Uint64("first-block-height", uint64(firstBlockHeight)),
		log.Uint64("last-block-height", uint64(lastBlockHeight)))

	s.metrics.blocksRate.Measure(int64(numHeaders))
	//for _, headerProof := range s.headers.HeaderWithProof {
	//	if !s.factory.conduit.drainAndCheckForShutdown(ctx) {
	//		return nil
	//	}
	//	//_, err := s.storage.ValidateHeaderForCommit(ctx, &services.ValidateHeaderForCommitInput{headerProof: headerProof})
	//	//
	//	//if err != nil {
	//	//	s.metrics.failedValidationBlocks.Inc()
	//	//	logger.Info("failed to validate header received via sync", log.Error(err), logfields.BlockHeight(headerProof.Header.BlockHeight())) // may be a valid failure if height isn't the next height
	//	//	break
	//	//}
	//
	//}

	if !s.factory.conduit.drainAndCheckForShutdown(ctx) {
		return nil
	}

	return s.factory.CreateCollectingAvailabilityResponseState()
}
