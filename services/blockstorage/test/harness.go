package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
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

func (d *harness) withSyncBroadcast(times int) *harness {
	d.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(times)
	return d
}

func (d *harness) withCommitStateDiff(times int) *harness {
	d.stateStorage.When("CommitStateDiff", mock.Any, mock.Any).Call(func (ctx context.Context, input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
		return &services.CommitStateDiffOutput{
			NextDesiredBlockHeight: input.ResultsBlockHeader.BlockHeight() + 1,
		}, nil
	}).Times(times)
	return d
}

func (d *harness) withValidateConsensusAlgos(times int) *harness {
	out := &handlers.HandleBlockConsensusOutput{}

	d.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Return(out, nil).Times(times)
	return d
}

func (d *harness) expectCommitStateDiffTimes(times int) {
	csdOut := &services.CommitStateDiffOutput{}

	d.stateStorage.When("CommitStateDiff", mock.Any, mock.Any).Return(csdOut, nil).Times(times)
}

func (d *harness) verifyMocks(t *testing.T, times int) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), d.gossip, d.stateStorage, d.consensus)
	require.NoError(t, err)
}

func (d *harness) commitBlock(ctx context.Context, blockPairContainer *protocol.BlockPairContainer) (*services.CommitBlockOutput, error) {
	return d.blockStorage.CommitBlock(ctx, &services.CommitBlockInput{
		BlockPair: blockPairContainer,
	})
}

func (d *harness) numOfWrittenBlocks() int {
	numBlocks, err := d.storageAdapter.GetNumBlocks()
	if err != nil {
		panic(err)
	}
	return int(numBlocks)
}

func (d *harness) getLastBlockHeight(ctx context.Context, t *testing.T) *services.GetLastCommittedBlockHeightOutput {
	out, err := d.blockStorage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})

	require.NoError(t, err)
	return out
}

func (d *harness) getBlock(height int) *protocol.BlockPairContainer {
	txBlock, err := d.storageAdapter.GetTransactionsBlock(primitives.BlockHeight(height))
	if err != nil {
		panic(err)
	}

	rxBlock, err := d.storageAdapter.GetResultsBlock(primitives.BlockHeight(height))
	if err != nil {
		panic(err)
	}

	return &protocol.BlockPairContainer{
		TransactionsBlock: txBlock,
		ResultsBlock:      rxBlock,
	}
}

func (d *harness) withSyncNoCommitTimeout(duration time.Duration) *harness {
	d.config.(*configForBlockStorageTests).syncNoCommit = duration
	return d
}

func (d *harness) withSyncCollectResponsesTimeout(duration time.Duration) *harness {
	d.config.(*configForBlockStorageTests).syncCollectResponses = duration
	return d
}

func (d *harness) withSyncCollectChunksTimeout(duration time.Duration) *harness {
	d.config.(*configForBlockStorageTests).syncCollectChunks = duration
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

func (d *harness) commitSomeBlocks(ctx context.Context, count int) {
	for i := 1; i <= count; i++ {
		d.commitBlock(ctx, builders.BlockPair().WithHeight(primitives.BlockHeight(i)).Build())
	}
}

func (d *harness) setupCustomBlocksForInit() time.Time {
	now := time.Now()
	for i := 1; i <= 10; i++ {
		now = now.Add(1 * time.Millisecond)
		d.storageAdapter.WriteNextBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(i)).WithBlockCreated(now).Build())
	}

	out := &handlers.HandleBlockConsensusOutput{}

	d.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Return(out, nil).Times(1)

	return now
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

func newBlockStorageHarness() *harness {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	keyPair := keys.Ed25519KeyPairForTests(0)
	cfg := createConfig(keyPair.PublicKey())

	d := &harness{config: cfg, logger: logger}
	d.stateStorage = &services.MockStateStorage{}
	d.storageAdapter = adapter.NewInMemoryBlockPersistence()

	d.consensus = &handlers.MockConsensusBlocksHandler{}

	// TODO: this might create issues with some tests later on, should move it to behavior or some other means of setup
	// Always expect at least 0 because sometimes it gets triggered because of the timings
	// HandleBlockConsensus always gets called when we try to start the sync which happens automatically
	d.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Return(nil, nil).AtLeast(0)

	d.gossip = &gossiptopics.MockBlockSync{}
	d.gossip.When("RegisterBlockSyncHandler", mock.Any).Return().Times(1)

	d.txPool = &services.MockTransactionPool{}
	// TODO: this might create issues with some tests later on, should move it to behavior or some other means of setup
	d.txPool.When("CommitTransactionReceipts", mock.Any, mock.Any).Call(func (ctx context.Context, input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight: input.ResultsBlockHeader.BlockHeight() + 1,
		}, nil
	}).AtLeast(0)

	return d
}

func (d *harness) start(ctx context.Context) *harness {
	registry := metric.NewRegistry()

	d.blockStorage = blockstorage.NewBlockStorage(ctx, d.config, d.storageAdapter, d.stateStorage, d.gossip, d.txPool, d.logger, registry, nil)
	d.blockStorage.RegisterConsensusBlocksHandler(d.consensus)

	return d
}
