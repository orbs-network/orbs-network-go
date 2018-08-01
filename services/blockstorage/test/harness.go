package test

import (
	. "github.com/onsi/gomega"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

type testConfig struct {
}

func (c *testConfig) BlockSyncCommitTimeout() time.Duration {
	return 70 * time.Millisecond
}

func (c *testConfig) NodeId() string {
	return "node1"
}

type driver struct {
	stateStorage   *services.MockStateStorage
	storageAdapter adapter.InMemoryBlockPersistence
	blockStorage   services.BlockStorage
}

func (d *driver) expectCommitStateDiff() {
	csdOut := &services.CommitStateDiffOutput{}

	d.stateStorage.When("CommitStateDiff", mock.Any).Return(csdOut, nil).Times(1)

}

func (d *driver) verifyMocks() {
	_, err := d.stateStorage.Verify()
	Expect(err).ToNot(HaveOccurred())
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
	Expect(err).ToNot(HaveOccurred())
	return out
}

func (d *driver) getBlock(height int) *protocol.BlockPairContainer {
	return d.storageAdapter.ReadAllBlocks()[height-1]
}

func testBlockStorageConfig() blockstorage.Config {
	return &testConfig{}
}

func (d *driver) failNextBlocks() {
	d.storageAdapter.FailNextBlocks()
}

func NewDriver() *driver {
	d := &driver{}
	d.stateStorage = &services.MockStateStorage{}
	d.storageAdapter = adapter.NewInMemoryBlockPersistence(&testConfig{})
	d.blockStorage = blockstorage.NewBlockStorage(testBlockStorageConfig(), d.storageAdapter, d.stateStorage, instrumentation.GetLogger())

	return d
}
