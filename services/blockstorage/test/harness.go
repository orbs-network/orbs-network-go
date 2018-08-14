package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

type driver struct {
	stateStorage   *services.MockStateStorage
	storageAdapter adapter.InMemoryBlockPersistence
	blockStorage   services.BlockStorage
	t              *testing.T
}

func (d *driver) expectCommitStateDiff() {
	csdOut := &services.CommitStateDiffOutput{}

	d.stateStorage.When("CommitStateDiff", mock.Any).Return(csdOut, nil).Times(1)
}

func (d *driver) verifyMocks() {
	_, err := d.stateStorage.Verify()
	require.NoError(d.t, err)
}

func (d *driver) commitBlock(blockPairContainer *protocol.BlockPairContainer) (*services.CommitBlockOutput, error) {
	return d.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPairContainer,
	})
}

func (d *driver) numOfWrittenBlocks() int {
	return len(d.storageAdapter.ReadAllBlocks())
}

func (d *driver) getLastBlockHeight() *services.GetLastCommittedBlockHeightOutput {
	out, err := d.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})

	require.NoError(d.t, err)
	return out
}

func (d *driver) getBlock(height int) *protocol.BlockPairContainer {
	return d.storageAdapter.ReadAllBlocks()[height-1]
}

func (d *driver) failNextBlocks() {
	d.storageAdapter.FailNextBlocks()
}

func NewDriver(t *testing.T) *driver {
	d := &driver{t: t}
	d.stateStorage = &services.MockStateStorage{}
	d.storageAdapter = adapter.NewInMemoryBlockPersistence()
	d.blockStorage = blockstorage.NewBlockStorage(config.NewBlockStorageConfig(70, 5, 5, 30*60), d.storageAdapter, d.stateStorage, instrumentation.GetLogger())

	return d
}
