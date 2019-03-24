// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateIdle_StaysIdleOnIdleReset(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		manualNoCommitTimer := synchronization.NewTimerWithManualTick()
		h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(log.DefaultTestingLogger(t), func() *synchronization.Timer {
			return manualNoCommitTimer
		})

		state := h.factory.CreateIdleState()
		nextState := h.processStateInBackgroundAndWaitUntilFinished(ctx, state, func() {
			h.factory.conduit <- idleResetMessage{}
			manualNoCommitTimer.ManualTick() // not required, added for completion (like in state_availability_requests_test)
		})

		require.IsType(t, &idleState{}, nextState, "nextState should still be idle")
		require.True(t, nextState != state, "processState state should be a different idle state (which was recreated so the timer starts from be beginning)")
	})
}

func TestStateIdle_MovesToCollectingAvailabilityResponsesOnNoCommitTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness(log.DefaultTestingLogger(t))

		state := h.factory.CreateIdleState()
		nextState := state.processState(ctx)

		require.IsType(t, &collectingAvailabilityResponsesState{}, nextState, "processState state should be collecting availability responses")
	})
}

func TestStateIdle_TerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	manualNoCommitTimer := synchronization.NewTimerWithManualTick()
	h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(log.DefaultTestingLogger(t), func() *synchronization.Timer {
		return manualNoCommitTimer
	})

	cancel()
	state := h.factory.CreateIdleState()
	nextState := state.processState(ctx)

	require.Nil(t, nextState, "context termination should return a nil new state")
}
