// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type blockSyncConfigForTests struct {
	nodeAddress              primitives.NodeAddress
	batchSize                uint32
	noCommit                 time.Duration
	collectResponses         time.Duration
	collectChunks            time.Duration
	referenceDistance        time.Duration
	managementReferenceGrace time.Duration
	blocksOrder              gossipmessages.SyncBlocksOrder
	descendingActivationDate string
}

func (c *blockSyncConfigForTests) NodeAddress() primitives.NodeAddress {
	return c.nodeAddress
}

func (c *blockSyncConfigForTests) BlockSyncNumBlocksInBatch() uint32 {
	return c.batchSize
}

func (c *blockSyncConfigForTests) BlockSyncNoCommitInterval() time.Duration {
	return c.noCommit
}

func (c *blockSyncConfigForTests) BlockSyncCollectResponseTimeout() time.Duration {
	return c.collectResponses
}

func (c *blockSyncConfigForTests) BlockSyncCollectChunksTimeout() time.Duration {
	return c.collectChunks
}

func (c *blockSyncConfigForTests) BlockSyncReferenceMaxAllowedDistance() time.Duration {
	return c.referenceDistance
}

func (c *blockSyncConfigForTests) ManagementReferenceGraceTimeout() time.Duration {
	return c.managementReferenceGrace
}

func (c *blockSyncConfigForTests) BlockSyncBlocksOrder() gossipmessages.SyncBlocksOrder {
	return c.blocksOrder
}

func (c *blockSyncConfigForTests) BlockSyncDescendingActivationDate() string {
	return c.descendingActivationDate
}

func newDefaultBlockSyncConfigForTests() *blockSyncConfigForTests {
	return &blockSyncConfigForTests{
		nodeAddress:              testKeys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress(),
		batchSize:                10,
		noCommit:                 3 * time.Millisecond,
		collectResponses:         3 * time.Millisecond,
		collectChunks:            3 * time.Millisecond,
		referenceDistance:        100 * time.Second,
		managementReferenceGrace: 100 * time.Second,
		blocksOrder:              gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING,
		descendingActivationDate: time.Now().AddDate(0, -1, 0).Format(time.RFC3339),//"2220-06-15T12:00:00.000Z",
	}
}

type blockSyncHarness struct {
	factory       *stateFactory
	config        *blockSyncConfigForTests
	gossip        *gossiptopics.MockBlockSync
	storage       *blockSyncStorageMock
	logger        log.Logger
	metricFactory metric.Factory
}

func newBlockSyncHarnessWithTimers(
	logger log.Logger,
	createCollectTimeoutTimer func() *synchronization.Timer,
	createNoCommitTimeoutTimer func() *synchronization.Timer,
	createWaitForChunksTimeoutTimer func() *synchronization.Timer,
) *blockSyncHarness {

	cfg := newDefaultBlockSyncConfigForTests()
	gossip := &gossiptopics.MockBlockSync{}
	storage := &blockSyncStorageMock{}
	conduit := make(blockSyncConduit)
	management := &services.MockManagement{}
	metricFactory := metric.NewRegistry()

	return &blockSyncHarness{
		logger:        logger,
		factory:       NewStateFactoryWithTimers(cfg, gossip, storage, conduit, management, cfg.blocksOrder, createCollectTimeoutTimer, createNoCommitTimeoutTimer, createWaitForChunksTimeoutTimer, logger, metricFactory),
		config:        cfg,
		gossip:        gossip,
		storage:       storage,
		metricFactory: metricFactory,
	}
}

func newBlockSyncHarness(logger log.Logger) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(logger, nil, nil, nil)
}

func newBlockSyncHarnessWithCollectResponsesTimer(logger log.Logger, createTimer func() *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(logger, createTimer, nil, nil)
}

func newBlockSyncHarnessWithManualNoCommitTimeoutTimer(logger log.Logger, createTimer func() *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(logger, nil, createTimer, nil)
}

func newBlockSyncHarnessWithManualWaitForChunksTimeoutTimer(logger log.Logger, createTimer func() *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(logger, nil, nil, createTimer)
}

func (h *blockSyncHarness) withWaitForChunksTimeout(d time.Duration) *blockSyncHarness {
	h.config.collectChunks = d
	return h
}

func (h *blockSyncHarness) withNodeAddress(address primitives.NodeAddress) *blockSyncHarness {
	h.config.nodeAddress = address
	return h
}

func (h *blockSyncHarness) withBatchSize(size uint32) *blockSyncHarness {
	h.config.batchSize = size
	return h
}

func (h *blockSyncHarness) withReferenceDistance(d time.Duration) *blockSyncHarness {
	h.config.referenceDistance = d
	return h
}

func (h *blockSyncHarness) withSyncBlocksOrder(order gossipmessages.SyncBlocksOrder) *blockSyncHarness {
	h.factory.syncBlocksOrder = order
	return h
}

func (h *blockSyncHarness) setSyncBlocksOrder(order gossipmessages.SyncBlocksOrder) {
	h.config.blocksOrder = order
}

func (h *blockSyncHarness) expectSyncOnStart() {
	h.expectUpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(10)
	h.expectBroadcastOfBlockAvailabilityRequest()
}

func (h *blockSyncHarness) eventuallyVerifyMocks(t *testing.T, times int) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), h.gossip)
	require.NoError(t, err)
}

func (h *blockSyncHarness) consistentlyVerifyMocks(t *testing.T, times int, message string) {
	err := test.ConsistentlyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), h.gossip)
	require.NoError(t, err, message)
}

func (h *blockSyncHarness) verifyMocks(t *testing.T) {
	ok, err := mock.VerifyMocks(h.storage, h.gossip)
	require.NoError(t, err)
	require.True(t, ok)
}

func (h *blockSyncHarness) processStateInBackgroundAndWaitUntilFinished(ctx context.Context, state syncState, whileStateIsProcessing func()) syncState {
	var nextState syncState
	stateProcessingFinished := make(chan bool)
	go func() {
		nextState = state.processState(ctx)
		stateProcessingFinished <- true
	}()
	whileStateIsProcessing()
	<-stateProcessingFinished
	return nextState
}

func (h *blockSyncHarness) expectLastCommittedBlockHeightQueryFromStorage(expectedHeight int) {
	out := &services.GetLastCommittedBlockHeightOutput{
		LastCommittedBlockHeight:    primitives.BlockHeight(expectedHeight),
		LastCommittedBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
	}
	h.storage.When("GetLastCommittedBlockHeight", mock.Any, mock.Any).Return(out, nil).Times(1)
}

func (h *blockSyncHarness) expectUpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(expectedHeight int) {
	h.storage.When("UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence", mock.Any).Times(1)
	h.expectLastCommittedBlockHeightQueryFromStorage(expectedHeight)
}

func (h *blockSyncHarness) expectBroadcastOfBlockAvailabilityRequestToFail() {
	h.storage.When("GetSyncState").Return(nil).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, errors.New("gossip failure")).Times(1)
}

func (h *blockSyncHarness) expectBroadcastOfBlockAvailabilityRequest() {
	h.storage.When("GetSyncState").Return(nil).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)
}

func (h *blockSyncHarness) verifyBroadcastOfBlockAvailabilityRequest(t *testing.T) {
	require.NoError(t, test.EventuallyVerify(50*time.Millisecond, h.gossip), "broadcast should be sent")
}

func (h *blockSyncHarness) expectBlockValidationQueriesFromStorage(numExpectedBlocks int) {
	h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Return(nil, nil).Times(numExpectedBlocks)
}

func (h *blockSyncHarness) expectBlockValidationQueriesFromStorageAndFailLastValidation(numExpectedBlocks int, expectedFirstBlockHeight primitives.BlockHeight) {
	h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
		if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(expectedFirstBlockHeight + primitives.BlockHeight(numExpectedBlocks-1)) {
			return nil, errors.Errorf("failed to validate Block #%d", numExpectedBlocks)
		}
		return nil, nil
	}).Times(numExpectedBlocks)
}

func (h *blockSyncHarness) expectBlockCommitsToStorage(numExpectedBlocks int) {
	outCommit := &services.CommitBlockOutput{}
	h.storage.When("NodeSyncCommitBlock", mock.Any, mock.Any).Return(outCommit, nil).Times(numExpectedBlocks)
}

func (h *blockSyncHarness) expectBlockCommitsToStorageAndFailLastCommit(numExpectedBlocks int, expectedFirstBlockHeight primitives.BlockHeight) {
	h.storage.When("NodeSyncCommitBlock", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
		if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(expectedFirstBlockHeight + primitives.BlockHeight(numExpectedBlocks-1)) {
			return nil, errors.Errorf("failed to commit Block #%d", numExpectedBlocks)
		}
		return nil, nil
	}).Times(numExpectedBlocks)
}

func (h *blockSyncHarness) expectSendingOfBlockSyncRequest() {
	h.storage.When("GetSyncState").Return(nil).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)
}

func (h *blockSyncHarness) expectSendingOfBlockSyncRequestToFail() {
	h.storage.When("GetSyncState").Return(nil).Times(1)
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, errors.New("gossip failure")).Times(1)
}
