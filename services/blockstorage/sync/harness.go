package sync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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

	return &blockSyncHarness{
		logger:    logger,
		sf:        NewStateFactory(cfg, gossip, storage, logger),
		ctx:       ctx,
		ctxCancel: cancel,
		config:    cfg,
		gossip:    gossip,
		storage:   storage,
	}
}

func (h *blockSyncHarness) withNodeKey(key primitives.Ed25519PublicKey) *blockSyncHarness {
	h.config.pk = key
	return h
}

func (h *blockSyncHarness) withNoCommitTimeout(duration time.Duration) *blockSyncHarness {
	h.config.noCommit = duration
	return h
}

func (h *blockSyncHarness) withBatchSize(size uint32) *blockSyncHarness {
	h.config.batchSize = size
	return h
}

func (h *blockSyncHarness) cancel() {
	h.ctxCancel()
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
