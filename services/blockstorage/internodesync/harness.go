package internodesync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type blockSyncConfigForTests struct {
	pk               primitives.Ed25519PublicKey
	batchSize        uint32
	noCommit         time.Duration
	collectResponses time.Duration
	collectChunks    time.Duration
}

func (c *blockSyncConfigForTests) NodePublicKey() primitives.Ed25519PublicKey {
	return c.pk
}

func (c *blockSyncConfigForTests) BlockSyncBatchSize() uint32 {
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

func newDefaultBlockSyncConfigForTests() *blockSyncConfigForTests {
	return &blockSyncConfigForTests{
		pk:               keys.Ed25519KeyPairForTests(1).PublicKey(),
		batchSize:        10,
		noCommit:         3 * time.Millisecond,
		collectResponses: 3 * time.Millisecond,
		collectChunks:    3 * time.Millisecond,
	}
}

func newBlockSyncConfigForTestsWithInfiniteTimeouts() *blockSyncConfigForTests {
	return &blockSyncConfigForTests{
		pk:               keys.Ed25519KeyPairForTests(1).PublicKey(),
		batchSize:        10,
		noCommit:         3 * time.Hour,
		collectResponses: 3 * time.Hour,
		collectChunks:    3 * time.Hour,
	}
}

type blockSyncHarness struct {
	factory       *stateFactory
	config        *blockSyncConfigForTests
	gossip        *gossiptopics.MockBlockSync
	storage       *blockSyncStorageMock
	logger        log.BasicLogger
	metricFactory metric.Factory
}

func newBlockSyncHarnessWithTimers(
	createCollectTimeoutTimer func() *synchronization.Timer,
	createNoCommitTimeoutTimer func() *synchronization.Timer,
	createWaitForChunksTimeoutTimer func() *synchronization.Timer,
) *blockSyncHarness {

	cfg := newDefaultBlockSyncConfigForTests()
	gossip := &gossiptopics.MockBlockSync{}
	storage := &blockSyncStorageMock{}
	logger := log.GetLogger()
	conduit := &blockSyncConduit{
		idleReset: make(chan struct{}),
		responses: make(chan *gossipmessages.BlockAvailabilityResponseMessage),
		blocks:    make(chan *gossipmessages.BlockSyncResponseMessage),
	}
	metricFactory := metric.NewRegistry()

	return &blockSyncHarness{
		logger:        logger,
		factory:       NewStateFactoryWithTimers(cfg, gossip, storage, conduit, createCollectTimeoutTimer, createNoCommitTimeoutTimer, createWaitForChunksTimeoutTimer, logger, metricFactory),
		config:        cfg,
		gossip:        gossip,
		storage:       storage,
		metricFactory: metricFactory,
	}
}

func newBlockSyncHarness() *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(nil, nil, nil)
}

func newBlockSyncHarnessWithCollectResponsesTimer(createTimer func() *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(createTimer, nil, nil)
}

func newBlockSyncHarnessWithManualNoCommitTimeoutTimer(createTimer func() *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(nil, createTimer, nil)
}

func newBlockSyncHarnessWithManualWaitForChunksTimeoutTimer(createTimer func() *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(nil, nil, createTimer)
}

func (h *blockSyncHarness) waitForShutdown(bs *BlockSync) bool {
	return test.Eventually(test.EVENTUALLY_LOCAL_E2E_TIMEOUT, func() bool {
		return bs.currentState == nil
	})
}

func (h *blockSyncHarness) waitForState(bs *BlockSync, desiredState syncState) bool {
	return test.Eventually(test.EVENTUALLY_LOCAL_E2E_TIMEOUT, func() bool {
		return bs.currentState != nil && bs.currentState.name() == desiredState.name()
	})
}

func (h *blockSyncHarness) withNodeKey(key primitives.Ed25519PublicKey) *blockSyncHarness {
	h.config.pk = key
	return h
}

func (h *blockSyncHarness) withBatchSize(size uint32) *blockSyncHarness {
	h.config.batchSize = size
	return h
}

func (h *blockSyncHarness) expectSyncOnStart() {
	h.expectPreSynchronizationUpdateOfConsensusAlgos(10)
	h.expectBroadcastOfBlockAvailabilityRequest()
}

func (h *blockSyncHarness) eventuallyVerifyMocks(t *testing.T, times int) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), h.gossip, h.storage)
	require.NoError(t, err)
}

func (h *blockSyncHarness) consistentlyVerifyMocks(t *testing.T, times int) {
	err := test.ConsistentlyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), h.gossip, h.storage)
	require.NoError(t, err)
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

func (h *blockSyncHarness) expectPreSynchronizationUpdateOfConsensusAlgos(expectedHeight int) {
	h.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock", mock.Any).Times(1)
	h.expectLastCommittedBlockHeightQueryFromStorage(expectedHeight)
}

func (h *blockSyncHarness) expectBroadcastOfBlockAvailabilityRequestToFail() {
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, errors.New("gossip failure")).Times(1)
}

func (h *blockSyncHarness) expectBroadcastOfBlockAvailabilityRequest() {
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)
}

func (h *blockSyncHarness) verifyBroadcastOfBlockAvailabilityRequest(t *testing.T) {
	require.NoError(t, test.EventuallyVerify(10*time.Millisecond, h.gossip), "broadcast should be sent")
}

func (h *blockSyncHarness) expectBlockValidationQueriesFromStorage(numExpectedBlocks int) {
	h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Return(nil, nil).Times(numExpectedBlocks)
}

func (h *blockSyncHarness) expectBlockValidationQueriesFromStorageAndFailLastValidation(numExpectedBlocks int, expectedFirstBlockHeight primitives.BlockHeight) {
	h.storage.When("ValidateBlockForCommit", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
		if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(expectedFirstBlockHeight + primitives.BlockHeight(numExpectedBlocks-1)) {
			return nil, errors.Errorf("failed to validate block #%d", numExpectedBlocks)
		}
		return nil, nil
	}).Times(numExpectedBlocks)
}

func (h *blockSyncHarness) expectBlockCommitsToStorage(numExpectedBlocks int) {
	outCommit := &services.CommitBlockOutput{}
	h.storage.When("CommitBlock", mock.Any, mock.Any).Return(outCommit, nil).Times(numExpectedBlocks)
}

func (h *blockSyncHarness) expectBlockCommitsToStorageAndFailLastCommit(numExpectedBlocks int, expectedFirstBlockHeight primitives.BlockHeight) {
	h.storage.When("CommitBlock", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
		if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(expectedFirstBlockHeight + primitives.BlockHeight(numExpectedBlocks-1)) {
			return nil, errors.Errorf("failed to commit block #%d", numExpectedBlocks)
		}
		return nil, nil
	}).Times(numExpectedBlocks)
}

func (h *blockSyncHarness) expectSendingOfBlockSyncRequest() {
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)
}

func (h *blockSyncHarness) expectSendingOfBlockSyncRequestToFail() {
	h.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, errors.New("gossip failure")).Times(1)
}
