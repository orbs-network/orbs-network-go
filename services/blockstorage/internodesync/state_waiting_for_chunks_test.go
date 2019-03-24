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
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStateWaitingForChunks_MovesToIdleOnTransportError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness(log.DefaultTestingLogger(t))

		h.expectLastCommittedBlockHeightQueryFromStorage(0)
		h.expectSendingOfBlockSyncRequestToFail()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
		nextState := state.processState(ctx)

		require.IsType(t, &idleState{}, nextState, "expecting back to idle on transport error")
		h.verifyMocks(t)
	})
}

func TestStateWaitingForChunks_MovesToIdleOnTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness(log.DefaultTestingLogger(t))

		h.expectLastCommittedBlockHeightQueryFromStorage(0)
		h.expectSendingOfBlockSyncRequest()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
		nextState := state.processState(ctx)

		require.IsType(t, &idleState{}, nextState, "expecting back to idle on timeout")
		h.verifyMocks(t)
	})
}

func TestStateWaitingForChunks_AcceptsNewBlockAndMovesToProcessingBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		manualWaitForChunksTimer := synchronization.NewTimerWithManualTick()
		blocksMessage := builders.BlockSyncResponseInput().Build().Message
		h := newBlockSyncHarnessWithManualWaitForChunksTimeoutTimer(log.DefaultTestingLogger(t), func() *synchronization.Timer {
			return manualWaitForChunksTimer
		}).withNodeAddress(blocksMessage.Sender.SenderNodeAddress())

		h.expectLastCommittedBlockHeightQueryFromStorage(10)
		h.expectSendingOfBlockSyncRequest()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
		nextState := h.processStateInBackgroundAndWaitUntilFinished(ctx, state, func() {
			h.factory.conduit <- blocksMessage
			manualWaitForChunksTimer.ManualTick() // not required, added for completion (like in state_availability_requests_test)
		})

		require.IsType(t, &processingBlocksState{}, nextState, "expecting to be at processing state after blocks arrived")
		pbs := nextState.(*processingBlocksState)
		require.NotNil(t, pbs.blocks, "blocks payload initialized in processing stage")
		require.Equal(t, blocksMessage.Sender, pbs.blocks.Sender, "expected sender in source message to be the same in the state")
		require.Equal(t, len(blocksMessage.BlockPairs), len(pbs.blocks.BlockPairs), "expected same number of blocks in message->state")
		require.Equal(t, blocksMessage.SignedChunkRange, pbs.blocks.SignedChunkRange, "expected signed range to be the same in message -> state")

		h.verifyMocks(t)
	})
}

func TestStateWaitingForChunks_TerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	manualWaitForChunksTimer := synchronization.NewTimerWithManualTick()
	h := newBlockSyncHarnessWithManualWaitForChunksTimeoutTimer(log.DefaultTestingLogger(t), func() *synchronization.Timer {
		return manualWaitForChunksTimer
	})

	h.expectLastCommittedBlockHeightQueryFromStorage(10)
	h.expectSendingOfBlockSyncRequest()

	cancel()
	state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
	nextState := state.processState(ctx)

	require.Nil(t, nextState, "context terminated, expected nil state")
}

func TestStateWaitingForChunks_RecoversFromByzantineMessageSource(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		differentThanStateSourceAddress := keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()
		byzantineBlocksMessage := builders.BlockSyncResponseInput().WithSenderNodeAddress(differentThanStateSourceAddress).Build().Message
		stateSourceAddress := keys.EcdsaSecp256K1KeyPairForTests(8).NodeAddress()
		validBlocksMessage := builders.BlockSyncResponseInput().WithSenderNodeAddress(stateSourceAddress).Build().Message
		h := newBlockSyncHarness(log.DefaultTestingLogger(t)).
			withNodeAddress(stateSourceAddress).
			withWaitForChunksTimeout(time.Second) // this is infinity when it comes to this test, it should timeout on a deadlock if it takes more than a sec to get the chunks

		h.expectLastCommittedBlockHeightQueryFromStorage(10)
		h.expectSendingOfBlockSyncRequest()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())
		nextState := h.processStateInBackgroundAndWaitUntilFinished(ctx, state, func() {
			h.factory.conduit <- byzantineBlocksMessage
			h.factory.conduit <- validBlocksMessage
		})

		require.IsType(t, &processingBlocksState{}, nextState, "expecting to move to the processing state even though a byzantine message arrived in the flow")
		h.verifyMocks(t)
	})
}

func TestStateWaitingForChunks_ByzantineStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chunks byzantine stress test in short mode")
	}

	test.WithContext(func(ctx context.Context) {
		differentThanStateSourceAddress := keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()
		byzantineBlocksMessage := builders.BlockSyncResponseInput().WithSenderNodeAddress(differentThanStateSourceAddress).Build().Message
		stateSourceAddress := keys.EcdsaSecp256K1KeyPairForTests(8).NodeAddress()
		validBlocksMessage := builders.BlockSyncResponseInput().WithSenderNodeAddress(stateSourceAddress).Build().Message
		h := newBlockSyncHarness(log.DefaultTestingLogger(t)).
			withNodeAddress(stateSourceAddress).
			withWaitForChunksTimeout(5 * time.Second)

		h.expectLastCommittedBlockHeightQueryFromStorage(10)
		h.expectSendingOfBlockSyncRequest()

		state := h.factory.CreateWaitingForChunksState(h.config.NodeAddress())

		byzLoopCount := 0
		nextState := h.processStateInBackgroundAndWaitUntilFinished(ctx, state, func() {
			// flood it with byzantine messages (DOS vector)
			byzLoopDone := make(chan struct{})
			go func() {
				for {
					select {
					case h.factory.conduit <- byzantineBlocksMessage:
						byzLoopCount++
					case <-byzLoopDone:
						return
					}
				}
			}()

			// send a valid block message after enough time has passed
			time.Sleep(500 * time.Millisecond)
			h.factory.conduit <- validBlocksMessage

			// stop the byzantine loop
			byzLoopDone <- struct{}{}
		})

		require.IsType(t, &processingBlocksState{}, nextState, "expecting to move to the processing state even though a byzantine message was hammering the flow")
		h.logger.Info("loop finished", log.Int("byzantine-message-count", byzLoopCount))
		require.True(t, byzLoopCount > 1, "expected more than one byzantine message to be processed")
		h.verifyMocks(t)
	})
}
