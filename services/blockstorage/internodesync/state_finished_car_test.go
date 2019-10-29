// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateFinishedCollectingAvailabilityResponses_ReturnsToIdleWhenNoResponsesReceived(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger)

			state := h.factory.CreateFinishedCARState(nil)
			nextState := state.processState(ctx)

			require.IsType(t, &idleState{}, nextState, "next state should be idle")
		})
	})
}

func TestStateFinishedCollectingAvailabilityResponses_MovesToWaitingForChunks(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			h := newBlockSyncHarness(harness.Logger)
			response := builders.BlockAvailabilityResponseInput().Build().Message
			state := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})
			nextState := state.processState(ctx)

			require.IsType(t, &waitingForChunksState{}, nextState, "next state should be waiting for chunks")
		})
	})
}

func TestStateFinishedCollectingAvailabilityResponses_ContextTerminationFlow(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		ctx, cancel := context.WithCancel(context.Background())

		h := newBlockSyncHarness(harness.Logger)
		response := builders.BlockAvailabilityResponseInput().Build().Message
		state := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})

		cancel()
		shouldBeNil := state.processState(ctx)

		require.Nil(t, shouldBeNil, "context terminated, state should be nil")
	})
}
