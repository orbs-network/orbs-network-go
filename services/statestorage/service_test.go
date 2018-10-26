package statestorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	. "github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestMerkleTreeInitializedByState(t *testing.T) {
	WithContext(func(ctx context.Context) {

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

func TestMerkleEntryGenerator(t *testing.T) {

	k1, v1 := getMerkleEntry("c", primitives.Ripmd160Sha256("k"), []byte("v"))
	k2, v2 := getMerkleEntry("c", primitives.Ripmd160Sha256("k1"), []byte("v"))
	k3, v3 := getMerkleEntry("c1", primitives.Ripmd160Sha256("k"), []byte("v"))

	require.Len(t, k1, 32)
	require.Len(t, k2, 32)
	require.Len(t, k3, 32)
	require.Len(t, v1, 32)
	require.Len(t, v2, 32)
	require.Len(t, v3, 32)

	require.Equal(t, v1, v2)
	require.Equal(t, v1, v3)

	require.NotEqual(t, k1, k2)
	require.NotEqual(t, k1, k3)
	require.NotEqual(t, k2, k3)

	k1, v1 = getMerkleEntry("c", primitives.Ripmd160Sha256("k"), []byte("v1"))
	k2, v2 = getMerkleEntry("c", primitives.Ripmd160Sha256("k"), []byte("v2"))

	require.Len(t, k1, 32)
	require.Len(t, k2, 32)
	require.Len(t, v1, 32)
	require.Len(t, v2, 32)

	require.NotEqual(t, v1, v2)

	require.Equal(t, k1, k2)
}
