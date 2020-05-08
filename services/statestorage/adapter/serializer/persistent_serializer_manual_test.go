package serializer

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/filesystem"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/servicesync"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
	"time"
)

type localConfig struct {
	dir         string
	chainId     primitives.VirtualChainId
	networkType protocol.SignerNetworkType
}

func (l *localConfig) BlockStorageFileSystemDataDir() string {
	return l.dir
}

func (l *localConfig) BlockStorageFileSystemMaxBlockSizeInBytes() uint32 {
	return 64 * 1024 * 1024
}

func (l *localConfig) VirtualChainId() primitives.VirtualChainId {
	return l.chainId
}

func (l *localConfig) NetworkType() protocol.SignerNetworkType {
	return l.networkType
}

func TestServiceSyncManually(t *testing.T) {
	t.Skip("manual test")

	with.Context(func(ctx context.Context) {
		logger := log.GetLogger().WithFilters(log.DiscardAll())
		metricFactory := metric.NewRegistry()

		persistence, blockHeight := getBlockPersistence(t, logger, metricFactory)

		blockTracker := synchronization.NewBlockTracker(logger, 0, 10)
		inmemory := memory.NewStatePersistence(metricFactory)
		stateStorageConfig := config.ForStateStorageTest(5, 5, 100)
		stateStorage := statestorage.NewStateStorage(stateStorageConfig, inmemory, blockTracker, logger, metricFactory)
		committer := servicesync.NewStateStorageCommitter(stateStorage)

		servicesync.NewServiceBlockSync(ctx, logger, persistence, committer)

		syncStart := time.Now()
		test.Eventually(10*time.Minute, func() bool {
			inmemoryBlockHeight, _, _, _, _, _, err := inmemory.ReadMetadata()
			fmt.Println(fmt.Sprintf("stateBH/persistentBH %d/%d", inmemoryBlockHeight, blockHeight))
			fmt.Println("time elapsed", time.Since(syncStart).String())
			return err == nil && inmemoryBlockHeight >= blockHeight-100 // gets stuck on last 5 blocks for whatever reason
		})
		fmt.Println("sync time", time.Since(syncStart).String())

		dumpStart := time.Now()
		dump, err := NewStatePersistenceSerializer(inmemory).Dump()
		fmt.Println("dump time", time.Since(dumpStart).String())
		require.NoError(t, err)

		err = ioutil.WriteFile("./dump.bin", dump, 0644)
		require.NoError(t, err)
	})
}

func TestServiceSyncStartupManually(t *testing.T) {
	t.Skip("manual test")

	with.Context(func(ctx context.Context) {
		dump, err := ioutil.ReadFile("./dump.bin")
		require.NoError(t, err)

		inmemory, err := NewPersistenceDeserializer(metric.NewRegistry()).Deserialize(dump)
		require.NoError(t, err)

		fmt.Println(inmemory.(*memory.InMemoryStatePersistence).Dump())

		logger := log.GetLogger().WithFilters(log.DiscardAll())
		metricFactory := metric.NewRegistry()

		persistence, blockHeight := getBlockPersistence(t, logger, metricFactory)

		blockTracker := synchronization.NewBlockTracker(logger, 0, 10)
		stateStorageConfig := config.ForStateStorageTest(5, 5, 100)
		stateStorage := statestorage.NewStateStorage(stateStorageConfig, inmemory, blockTracker, logger, metricFactory)
		committer := servicesync.NewStateStorageCommitter(stateStorage)

		servicesync.NewServiceBlockSync(ctx, logger, persistence, committer)

		syncStart := time.Now()
		test.Eventually(10*time.Second, func() bool {
			inmemoryBlockHeight, _, _, _, _, _, err := inmemory.ReadMetadata()
			fmt.Println(fmt.Sprintf("stateBH/persistentBH %d/%d", inmemoryBlockHeight, blockHeight))
			fmt.Println("time elapsed", time.Since(syncStart).String())
			return err == nil && inmemoryBlockHeight >= blockHeight-100 // gets stuck on last 5 blocks for whatever reason
		})
		fmt.Println("sync time", time.Since(syncStart).String())
	})

}

func getBlockPersistence(t *testing.T, logger log.Logger, metricFactory metric.Factory) (*filesystem.BlockPersistence, primitives.BlockHeight) {
	start := time.Now()

	persistence, err := filesystem.NewBlockPersistence(&localConfig{
		chainId:     1100000,
		networkType: protocol.NETWORK_TYPE_RESERVED,
		dir:         "/usr/local/var/orbs",
	}, logger, metricFactory)

	fmt.Println("startup time", time.Since(start).String())

	if err != nil {
		t.Fatal("failed to start processing", err)
	}

	blockHeight, err := persistence.GetLastBlockHeight()
	if err != nil {
		t.Fatal("failed to get block height", err)
	}

	return persistence, blockHeight
}
