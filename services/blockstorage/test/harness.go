package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

type harness struct {
	stateStorage   *services.MockStateStorage
	storageAdapter adapter.InMemoryBlockPersistence
	blockStorage   services.BlockStorage
	consensus      *handlers.MockConsensusBlocksHandler
	gossip         *gossiptopics.MockBlockSync
	config         blockstorage.Config
	logger         log.BasicLogger
	ctx            context.Context
}

func (d *harness) expectCommitStateDiff() {
	d.expectCommitStateDiffTimes(1)
}

func (d *harness) expectCommitStateDiffTimes(times int) {
	csdOut := &services.CommitStateDiffOutput{}

	d.stateStorage.When("CommitStateDiff", mock.Any).Return(csdOut, nil).Times(times)
}

func (d *harness) expectValidateWithConsensusAlgosTimes(times int) {
	out := &handlers.HandleBlockConsensusOutput{}

	d.consensus.When("HandleBlockConsensus", mock.Any).Return(out, nil).Times(times)
}

func (d *harness) verifyMocks(t *testing.T) {
	err := test.EventuallyVerify(d.gossip, d.stateStorage, d.consensus)
	require.NoError(t, err)
}

func (d *harness) commitBlock(blockPairContainer *protocol.BlockPairContainer) (*services.CommitBlockOutput, error) {
	return d.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPairContainer,
	})
}

func (d *harness) numOfWrittenBlocks() int {
	return len(d.storageAdapter.ReadAllBlocks())
}

func (d *harness) getLastBlockHeight(t *testing.T) *services.GetLastCommittedBlockHeightOutput {
	out, err := d.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})

	require.NoError(t, err)
	return out
}

func (d *harness) getBlock(height int) *protocol.BlockPairContainer {
	return d.storageAdapter.ReadAllBlocks()[height-1]
}

func (d *harness) failNextBlocks() {
	d.storageAdapter.FailNextBlocks()
}

func (d *harness) setBatchSize(batchSize uint32) {
	d.config.(config.NodeConfig).SetUint32(config.BLOCK_SYNC_BATCH_SIZE, batchSize)
}

func newCustomSetupHarness(setup func(persistence adapter.InMemoryBlockPersistence, consensus *handlers.MockConsensusBlocksHandler)) *harness {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	keyPair := keys.Ed25519KeyPairForTests(0)

	cfg := config.EmptyConfig()
	cfg.SetNodePublicKey(keyPair.PublicKey())
	cfg.SetUint32(config.BLOCK_SYNC_BATCH_SIZE, 10000)

	cfg.SetDuration(config.BLOCK_SYNC_INTERVAL, 3*time.Millisecond)
	cfg.SetDuration(config.BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 1*time.Millisecond)
	cfg.SetDuration(config.BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT, 5*time.Second)

	cfg.SetDuration(config.BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START, 5*time.Second)
	cfg.SetDuration(config.BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END, 5*time.Second)
	cfg.SetDuration(config.BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW, 30*time.Minute)

	d := &harness{config: cfg, logger: logger}
	d.stateStorage = &services.MockStateStorage{}
	d.storageAdapter = adapter.NewInMemoryBlockPersistence()

	d.consensus = &handlers.MockConsensusBlocksHandler{}

	// Always expect at least 0 because sometimes it gets triggered because of the timings
	// HandleBlockConsensus always gets called when we try to start the sync which happens automatically
	d.consensus.When("HandleBlockConsensus", mock.Any).Return(nil, nil).AtLeast(0)

	if setup != nil {
		setup(d.storageAdapter, d.consensus)
	}

	d.gossip = &gossiptopics.MockBlockSync{}
	d.gossip.When("RegisterBlockSyncHandler", mock.Any).Return().Times(1)
	d.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(0)

	ctx := context.Background()
	d.blockStorage = blockstorage.NewBlockStorage(ctx, cfg, d.storageAdapter, d.stateStorage, d.gossip, logger)
	d.ctx = ctx

	d.blockStorage.RegisterConsensusBlocksHandler(d.consensus)

	return d
}

func newHarness() *harness {
	return newCustomSetupHarness(nil)
}
