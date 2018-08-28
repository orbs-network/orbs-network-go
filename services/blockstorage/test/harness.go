package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

type driver struct {
	stateStorage   *services.MockStateStorage
	storageAdapter adapter.InMemoryBlockPersistence
	blockStorage   services.BlockStorage
	gossip         *gossiptopics.MockBlockSync
	config         blockstorage.Config
	logger         log.BasicLogger
	ctx            context.Context
}

func (d *driver) expectCommitStateDiff() {
	d.expectCommitStateDiffTimes(1)
}

func (d *driver) expectCommitStateDiffTimes(times int) {
	csdOut := &services.CommitStateDiffOutput{}

	d.stateStorage.When("CommitStateDiff", mock.Any).Return(csdOut, nil).Times(times)
}

func (d *driver) verifyMocks(t *testing.T) {
	_, err := d.stateStorage.Verify()
	require.NoError(t, err)

	_, err = d.gossip.Verify()
	require.NoError(t, err)
}

func (d *driver) commitBlock(blockPairContainer *protocol.BlockPairContainer) (*services.CommitBlockOutput, error) {
	return d.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPairContainer,
	})
}

func (d *driver) numOfWrittenBlocks() int {
	return len(d.storageAdapter.ReadAllBlocks())
}

func (d *driver) getLastBlockHeight(t *testing.T) *services.GetLastCommittedBlockHeightOutput {
	out, err := d.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})

	require.NoError(t, err)
	return out
}

func (d *driver) getBlock(height int) *protocol.BlockPairContainer {
	return d.storageAdapter.ReadAllBlocks()[height-1]
}

func (d *driver) failNextBlocks() {
	d.storageAdapter.FailNextBlocks()
}

func (d *driver) setBatchSize(batchSize uint32) {
	d.config.(config.NodeConfig).SetUint32(config.BLOCK_SYNC_BATCH_SIZE, batchSize)
}

func NewCustomSetupDriver(setup func(persistence adapter.InMemoryBlockPersistence)) *driver {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	keyPair := keys.Ed25519KeyPairForTests(0)

	cfg := config.EmptyConfig()
	cfg.SetNodePublicKey(keyPair.PublicKey())
	cfg.SetUint32(config.BLOCK_SYNC_BATCH_SIZE, 10000)

	cfg.SetDuration(config.BLOCK_SYNC_INTERVAL, 3*time.Millisecond)
	cfg.SetDuration(config.BLOCK_SYNC_COLLECT_RESPONSE_TIMEOUT, 1*time.Millisecond)
	cfg.SetDuration(config.BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_START, 5*time.Second)
	cfg.SetDuration(config.BLOCK_TRANSACTION_RECEIPT_QUERY_GRACE_END, 5*time.Second)
	cfg.SetDuration(config.BLOCK_TRANSACTION_RECEIPT_QUERY_EXPIRATION_WINDOW, 30*time.Minute)

	d := &driver{config: cfg, logger: logger}
	d.stateStorage = &services.MockStateStorage{}
	d.storageAdapter = adapter.NewInMemoryBlockPersistence()

	if setup != nil {
		setup(d.storageAdapter)
	}

	d.gossip = &gossiptopics.MockBlockSync{}
	d.gossip.When("RegisterBlockSyncHandler", mock.Any).Return().Times(1)

	ctx := context.Background()
	d.blockStorage = blockstorage.NewBlockStorage(ctx, cfg, d.storageAdapter, d.stateStorage, d.gossip, logger)
	d.ctx = ctx

	d.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(0)

	return d
}

func NewDriver() *driver {
	return NewCustomSetupDriver(nil)
}
