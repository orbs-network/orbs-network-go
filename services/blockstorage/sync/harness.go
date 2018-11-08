package sync

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
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// the storage mock should be moved to its own file, but i have a weird goland bug where it will not identify it and its driving me mad, putting this here for now
type blockSyncStorageMock struct {
	mock.Mock
}

func (s *blockSyncStorageMock) GetLastCommittedBlockHeight(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
	ret := s.Called(ctx, input)
	if out := ret.Get(0); out != nil {
		return out.(*services.GetLastCommittedBlockHeightOutput), ret.Error(1)
	} else {
		return nil, ret.Error(1)
	}
}

func (s *blockSyncStorageMock) CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	ret := s.Called(ctx, input)
	if out := ret.Get(0); out != nil {
		return out.(*services.CommitBlockOutput), ret.Error(1)
	} else {
		return nil, ret.Error(1)
	}
}

func (s *blockSyncStorageMock) ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	ret := s.Called(ctx, input)
	if out := ret.Get(0); out != nil {
		return out.(*services.ValidateBlockForCommitOutput), ret.Error(1)
	} else {
		return nil, ret.Error(1)
	}
}

func (s *blockSyncStorageMock) UpdateConsensusAlgosAboutLatestCommittedBlock(ctx context.Context) {
	s.Called(ctx)
}

// end of storage mock

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

type blockSyncHarness struct {
	factory       *stateFactory
	ctx           context.Context
	config        *blockSyncConfigForTests
	gossip        *gossiptopics.MockBlockSync
	storage       *blockSyncStorageMock
	logger        log.BasicLogger
	ctxCancel     context.CancelFunc
	metricFactory metric.Factory
}

func newBlockSyncHarnessWithTimers(
	explicitCollectTimeoutTimer *synchronization.Timer,
	explicitNoCommitTimeoutTimer *synchronization.Timer,
	explicitWaitForChunksTimeoutTimer *synchronization.Timer,
) *blockSyncHarness {

	cfg := newDefaultBlockSyncConfigForTests()
	gossip := &gossiptopics.MockBlockSync{}
	storage := &blockSyncStorageMock{}
	logger := log.GetLogger()
	ctx, cancel := context.WithCancel(context.Background())
	conduit := &blockSyncConduit{
		idleReset: make(chan struct{}),
		responses: make(chan *gossipmessages.BlockAvailabilityResponseMessage),
		blocks:    make(chan *gossipmessages.BlockSyncResponseMessage),
	}
	metricFactory := metric.NewRegistry()

	var createCollectTimeoutTimer func() *synchronization.Timer = nil
	if explicitCollectTimeoutTimer != nil {
		createCollectTimeoutTimer = func() *synchronization.Timer { return explicitCollectTimeoutTimer }
	}

	var createNoCommitTimeoutTimer func() *synchronization.Timer = nil
	if explicitNoCommitTimeoutTimer != nil {
		createNoCommitTimeoutTimer = func() *synchronization.Timer { return explicitNoCommitTimeoutTimer }
	}

	var createWaitForChunksTimeoutTimer func() *synchronization.Timer = nil
	if explicitWaitForChunksTimeoutTimer != nil {
		createWaitForChunksTimeoutTimer = func() *synchronization.Timer { return explicitWaitForChunksTimeoutTimer }
	}

	return &blockSyncHarness{
		logger:        logger,
		factory:       NewStateFactoryWithTimers(cfg, gossip, storage, conduit, createCollectTimeoutTimer, createNoCommitTimeoutTimer, createWaitForChunksTimeoutTimer, logger, metricFactory),
		ctx:           ctx,
		ctxCancel:     cancel,
		config:        cfg,
		gossip:        gossip,
		storage:       storage,
		metricFactory: metricFactory,
	}
}

func newBlockSyncHarness() *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(nil, nil, nil)
}

func newBlockSyncHarnessWithCollectResponsesTimer(manualTimer *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(manualTimer, nil, nil)
}

func newBlockSyncHarnessWithManualNoCommitTimeoutTimer(manualTimer *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(nil, manualTimer, nil)
}

func newBlockSyncHarnessWithManualWaitForChunksTimeoutTimer(manualTimer *synchronization.Timer) *blockSyncHarness {
	return newBlockSyncHarnessWithTimers(nil, nil, manualTimer)
}

func (h *blockSyncHarness) withCtxTimeout(d time.Duration) *blockSyncHarness {
	h.ctx, h.ctxCancel = context.WithTimeout(h.ctx, d)
	return h
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

func (h *blockSyncHarness) cancel() {
	h.ctxCancel()
}

func (h *blockSyncHarness) expectingSyncOnStart() {
	h.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock", mock.Any).Times(1)
	h.expectLastCommittedBlockHeight(primitives.BlockHeight(10))
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)
}

func (h *blockSyncHarness) eventuallyVerifyMocks(t *testing.T, times int) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), h.gossip, h.storage)
	require.NoError(t, err)
}

func (h *blockSyncHarness) verifyMocks(t *testing.T) {
	ok, err := mock.VerifyMocks(h.storage, h.gossip)
	require.NoError(t, err)
	require.True(t, ok)
}

func (h *blockSyncHarness) processStateAndWaitUntilFinished(state syncState, whileStateIsProcessing func()) syncState {
	var nextState syncState
	stateProcessingFinished := make(chan bool)
	go func() {
		nextState = state.processState(h.ctx)
		stateProcessingFinished <- true
	}()
	whileStateIsProcessing()
	<-stateProcessingFinished
	return nextState
}

func (h *blockSyncHarness) expectLastCommittedBlockHeight(expectedHeight primitives.BlockHeight) {
	out := &services.GetLastCommittedBlockHeightOutput{
		LastCommittedBlockHeight:    expectedHeight,
		LastCommittedBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
	}
	h.storage.When("GetLastCommittedBlockHeight", mock.Any, mock.Any).Return(out, nil).Times(1)
}
