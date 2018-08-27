package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

type InMemoryStatePersistence interface {
	adapter.StatePersistence
	Dump() string
	WaitUntilCommittedBlockOfHeight(height primitives.BlockHeight) error
}

type TestStatePersistence struct {
	inner                *adapter.InMemoryStatePersistence
	blockTrackerForTests *synchronization.BlockTracker
}

func NewInMemoryStatePersistence() InMemoryStatePersistence {
	return &TestStatePersistence{
		inner:                adapter.NewInMemoryStatePersistence(),
		blockTrackerForTests: synchronization.NewBlockTracker(0, 64000, time.Duration(1*time.Hour)),
	}
}

func (t *TestStatePersistence) WriteState(height primitives.BlockHeight, contractStateDiffs []*protocol.ContractStateDiff) error {
	err := t.inner.WriteState(height, contractStateDiffs)
	if err != nil {
		return err
	}

	t.blockTrackerForTests.IncrementHeight()
	return nil
}

func (t *TestStatePersistence) ReadState(height primitives.BlockHeight, contract primitives.ContractName) (map[string]*protocol.StateRecord, error) {
	return t.inner.ReadState(height, contract)
}

func (t *TestStatePersistence) WriteMerkleRoot(height primitives.BlockHeight, sha256 primitives.MerkleSha256) error {
	return t.inner.WriteMerkleRoot(height, sha256)
}

func (t *TestStatePersistence) ReadMerkleRoot(height primitives.BlockHeight) (primitives.MerkleSha256, error) {
	return t.inner.ReadMerkleRoot(height)
}

func (t *TestStatePersistence) Dump() string {
	return t.inner.Dump()
}

func (t *TestStatePersistence) WaitUntilCommittedBlockOfHeight(height primitives.BlockHeight) error {
	return t.blockTrackerForTests.WaitForBlock(height)
}
