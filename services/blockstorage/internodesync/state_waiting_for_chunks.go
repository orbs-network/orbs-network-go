// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type waitingForChunksState struct {
	factory           *stateFactory
	sourceNodeAddress primitives.NodeAddress
	client            *blockSyncClient
	createTimer       func() *synchronization.Timer
	logger            log.BasicLogger
	conduit           blockSyncConduit
	metrics           waitingStateMetrics
}

func (s *waitingForChunksState) name() string {
	return "waiting-for-chunks-state"
}

func (s *waitingForChunksState) String() string {
	return fmt.Sprintf("%s-from-source-%s", s.name(), s.sourceNodeAddress)
}

func (s *waitingForChunksState) processState(ctx context.Context) syncState {
	start := time.Now()
	defer s.metrics.timeSpentInState.RecordSince(start) // runtime metric
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	err := s.client.petitionerSendBlockSyncRequest(ctx, gossipmessages.BLOCK_TYPE_BLOCK_PAIR, s.sourceNodeAddress)
	if err != nil {
		logger.Info("could not request block chunk from source", log.Error(err), log.Stringable("source", s.sourceNodeAddress))

		return s.factory.CreateIdleState()
	}

	timeout := s.createTimer()
	for {
		select {
		case <-timeout.C:
			logger.Info("timed out when waiting for chunks", log.Stringable("source", s.sourceNodeAddress))
			s.metrics.timesTimeout.Inc()
			return s.factory.CreateIdleState()
		case e := <-s.conduit:
			switch blocks := e.(type) {
			case *gossipmessages.BlockSyncResponseMessage:
				if blocks.Sender.SenderNodeAddress().Equal(s.sourceNodeAddress) {
					logger.Info("got blocks from sync", log.Stringable("source", s.sourceNodeAddress))
					s.metrics.timesSuccessful.Inc()
					return s.factory.CreateProcessingBlocksState(blocks)
				} else { // we do not abort in this case, just keep waiting for the real message to come in
					logger.Info("byzantine message detected, expected source key does not match incoming",
						log.Stringable("source", s.sourceNodeAddress),
						log.Stringable("message-sender", blocks.Sender.SenderNodeAddress()))
					s.metrics.timesByzantine.Inc()
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}
