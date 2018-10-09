package sync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
)

// the storage mock should be moved to its own file, but i have a weird goland bug where it will not identify it and its driving me mad, putting this here for now
type blockSyncStorageMock struct {
	mock.Mock
}

func (s *blockSyncStorageMock) GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight) {
	ret := s.Called()
	return ret.Get(0).([]*protocol.BlockPairContainer), ret.Get(1).(primitives.BlockHeight), ret.Get(2).(primitives.BlockHeight)
}

func (s *blockSyncStorageMock) LastCommittedBlockHeight() primitives.BlockHeight {
	ret := s.Called()
	return ret.Get(0).(primitives.BlockHeight)
}

func (s *blockSyncStorageMock) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	ret := s.Called(input)
	return nil, ret.Error(0)
}

func (s *blockSyncStorageMock) ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	ret := s.Called(input)
	return nil, ret.Error(0)
}

func (s *blockSyncStorageMock) UpdateConsensusAlgosAboutLatestCommittedBlock() {
	s.Called()
}

// end of storage mock

type blockSyncHarness struct {
	sf        *stateFactory
	ctx       context.Context
	config    config.BlockStorageConfig
	gossip    *gossiptopics.MockBlockSync
	storage   *blockSyncStorageMock
	logger    log.BasicLogger
	ctxCancel context.CancelFunc
}

func newBlockSyncHarness() *blockSyncHarness {

	cfg := config.ForBlockStorageTests(keys.Ed25519KeyPairForTests(1).PublicKey())
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
	h.config = config.ForBlockStorageTests(key)
	h.sf = NewStateFactory(h.config, h.gossip, h.storage, h.logger)
	return h
}

//func (h *blockSyncHarness) withNoCommitTimeout(d time.Duration) *blockSyncHarness {
//	h.config.SetDuration(config.BLOCK_SYNC_INTERVAL, d)
//	h.sf = NewStateFactory(h.config, h.gossip, h.storage, h.logger)
//	return h
//}
//
//func (h *blockSyncHarness) withCollectResponseTimeout(d time.Duration) *blockSyncHarness {
//	h.config.SetDuration(config.BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, d)
//	h.sf = NewStateFactory(h.config, h.gossip, h.storage, h.logger)
//	return h
//}
//
//func (h *blockSyncHarness) withWaitForChunksTimeout(d time.Duration) *blockSyncHarness {
//	h.config.SetDuration(config.BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, d)
//	h.sf = NewStateFactory(h.config, h.gossip, h.storage, h.logger)
//	return h
//}

func (h *blockSyncHarness) cancel() {
	h.ctxCancel()
}

func (h *blockSyncHarness) verifyMocks(t *testing.T) {
	ok, err := mock.VerifyMocks(h.storage, h.gossip)
	require.NoError(t, err)
	require.True(t, ok)
}
