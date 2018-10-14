package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

type configForBlockStorageTests struct {
	pk                    primitives.Ed25519PublicKey
	syncBatchSize         uint32
	syncNoCommit          time.Duration
	syncCollectResponses  time.Duration
	syncCollectChunks     time.Duration
	queryGraceStart       time.Duration
	queryGraceEnd         time.Duration
	queryExpirationWindow time.Duration
}

func (c *configForBlockStorageTests) NodePublicKey() primitives.Ed25519PublicKey {
	return c.pk
}

func (c *configForBlockStorageTests) BlockSyncBatchSize() uint32 {
	return c.syncBatchSize
}

func (c *configForBlockStorageTests) BlockSyncNoCommitInterval() time.Duration {
	return c.syncNoCommit
}

func (c *configForBlockStorageTests) BlockSyncCollectResponseTimeout() time.Duration {
	return c.syncCollectResponses
}

func (c *configForBlockStorageTests) BlockSyncCollectChunksTimeout() time.Duration {
	return c.syncCollectChunks
}

func (c *configForBlockStorageTests) BlockTransactionReceiptQueryGraceStart() time.Duration {
	return c.queryGraceStart
}

func (c *configForBlockStorageTests) BlockTransactionReceiptQueryGraceEnd() time.Duration {
	return c.queryGraceEnd
}

func (c *configForBlockStorageTests) BlockTransactionReceiptQueryExpirationWindow() time.Duration {
	return c.queryExpirationWindow
}

type harness struct {
	stateStorage   *services.MockStateStorage
	storageAdapter adapter.InMemoryBlockPersistence
	blockStorage   services.BlockStorage
	consensus      *handlers.MockConsensusBlocksHandler
	gossip         *gossiptopics.MockBlockSync
	txPool         *services.MockTransactionPool
	config         config.BlockStorageConfig
	logger         log.BasicLogger
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

func (d *harness) verifyMocks(t *testing.T, times int) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), d.gossip, d.stateStorage, d.consensus)
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

func (d *harness) withSyncNoCommitTimeout(duration time.Duration) *harness {
	d.config.(*configForBlockStorageTests).syncNoCommit = duration
	return d
}

func (d *harness) withBatchSize(size uint32) *harness {
	d.config.(*configForBlockStorageTests).syncBatchSize = size
	return d
}

func (d *harness) withNodeKey(key primitives.Ed25519PublicKey) *harness {
	d.config.(*configForBlockStorageTests).pk = key
	return d
}

func (d *harness) failNextBlocks() {
	d.storageAdapter.FailNextBlocks()
}

func createConfig(nodePublicKey primitives.Ed25519PublicKey) config.BlockStorageConfig {
	cfg := &configForBlockStorageTests{}
	cfg.pk = nodePublicKey
	cfg.syncBatchSize = 2
	cfg.syncNoCommit = 30 * time.Second // setting a long time here so sync never starts during the tests
	cfg.syncCollectResponses = 5 * time.Millisecond
	cfg.syncCollectChunks = 20 * time.Millisecond

	cfg.queryGraceStart = 5 * time.Second
	cfg.queryGraceEnd = 5 * time.Second
	cfg.queryExpirationWindow = 30 * time.Minute

	return cfg
}

func (d *harness) setupSomeBlocks(count int) {
	d.expectCommitStateDiffTimes(count)

	for i := 1; i <= count; i++ {
		d.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(i)).Build())
	}
}

func newCustomSetupHarness(ctx context.Context, setup func(persistence adapter.InMemoryBlockPersistence, consensus *handlers.MockConsensusBlocksHandler)) *harness {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	keyPair := keys.Ed25519KeyPairForTests(0)
	cfg := createConfig(keyPair.PublicKey())

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

	d.txPool = &services.MockTransactionPool{}
	d.txPool.When("CommitTransactionReceipts", mock.Any).Return(nil, nil).AtLeast(0)

	d.blockStorage = blockstorage.NewBlockStorage(ctx, cfg, d.storageAdapter, d.stateStorage, d.gossip, d.txPool, logger)
	d.blockStorage.RegisterConsensusBlocksHandler(d.consensus)

	return d
}

func newHarness(ctx context.Context) *harness {
	return newCustomSetupHarness(ctx, nil)
}
