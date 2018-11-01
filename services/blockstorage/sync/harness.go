package sync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
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

func (s *blockSyncStorageMock) LastCommittedBlockHeight() primitives.BlockHeight {
	ret := s.Called()
	return ret.Get(0).(primitives.BlockHeight)
}

func (s *blockSyncStorageMock) CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	ret := s.Called(ctx, input)
	return nil, ret.Error(0)
}

func (s *blockSyncStorageMock) ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	ret := s.Called(ctx, input)
	return nil, ret.Error(0)
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

type blockSyncHarness struct {
	sf        *stateFactory
	ctx       context.Context
	config    *blockSyncConfigForTests
	gossip    *gossiptopics.MockBlockSync
	storage   *blockSyncStorageMock
	logger    log.BasicLogger
	ctxCancel context.CancelFunc
}

func newBlockSyncHarness() *blockSyncHarness {

	cfg := &blockSyncConfigForTests{
		pk:               keys.Ed25519KeyPairForTests(1).PublicKey(),
		batchSize:        10,
		noCommit:         3 * time.Millisecond,
		collectResponses: 3 * time.Millisecond,
		collectChunks:    3 * time.Millisecond,
	}
	gossip := &gossiptopics.MockBlockSync{}
	storage := &blockSyncStorageMock{}
	logger := log.GetLogger()
	ctx, cancel := context.WithCancel(context.Background())
	conduit := &blockSyncConduit{
		idleReset: make(chan struct{}),
		responses: make(chan *gossipmessages.BlockAvailabilityResponseMessage),
		blocks:    make(chan *gossipmessages.BlockSyncResponseMessage),
	}

	return &blockSyncHarness{
		logger:    logger,
		sf:        NewStateFactory(cfg, gossip, storage, conduit, logger),
		ctx:       ctx,
		ctxCancel: cancel,
		config:    cfg,
		gossip:    gossip,
		storage:   storage,
	}
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

func (h *blockSyncHarness) withNoCommitTimeout(duration time.Duration) *blockSyncHarness {
	h.config.noCommit = duration
	return h
}

func (h *blockSyncHarness) withWaitForChunksTimeout(duration time.Duration) *blockSyncHarness {
	h.config.collectChunks = duration
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
	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
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

func (h *blockSyncHarness) nextState(state syncState, trigger func()) syncState {
	var nextState syncState
	latch := make(chan struct{})
	go func() {
		nextState = state.processState(h.ctx)
		latch <- struct{}{}
	}()
	trigger()
	<-latch
	return nextState
}
