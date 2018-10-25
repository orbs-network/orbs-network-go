package statestorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestMerkleTreeInitializedByState(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		var contract primitives.ContractName = "c"
		k := primitives.Ripmd160Sha256("k")
		v := []byte("value")

		// prepare in memory state persistence with arbitrary state
		persistence := adapter.NewInMemoryStatePersistence()
		record := (&protocol.StateRecordBuilder{
			Key:   k,
			Value: v,
		}).Build()
		persistence.Write(0, 0, nil, adapter.ChainState{contract: adapter.ContractState{string(k): record}})

		// initialize a new state storage service
		cfg := config.ForStateStorageTest(5, 5, 100)
		logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
		service := NewStateStorage(cfg, persistence, logger)

		// check that the merkle root was initialized according to the
		serviceRoot, _ := service.GetStateHash(ctx, &services.GetStateHashInput{BlockHeight: 0})

		forest, root := merkle.NewForest()
		merkleK, merkleV := getMerkleEntry(contract, k, v)
		root, _ = forest.Update(root, merkle.MerkleDiffs{merkleK: merkleV})
		require.Equal(t, serviceRoot.StateRootHash, root)
	})
}
