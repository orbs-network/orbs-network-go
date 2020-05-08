package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter/serializer"
	"github.com/orbs-network/scribe/log"
	"io/ioutil"
	"path/filepath"
)

func loadStatePersistence(config config.NodeConfig, logger log.Logger, registry metric.Registry) *memory.InMemoryStatePersistence {
	stateFilePath := getStateFilePath(config)
	persistence := memory.NewStatePersistence(registry)

	dump, err := ioutil.ReadFile(stateFilePath)
	if err != nil {
		logger.Error("failed to read state file", log.Error(err))
		return persistence
	}

	if err := serializer.NewStatePersistenceDeserializer(persistence).Deserialize(dump); err != nil {
		logger.Error("failed to deserialize state file", log.Error(err))
	} else {
		logger.Info("finished loading the state")
	}

	return persistence
}

func shutdownStatePersistence(ctx context.Context, config config.NodeConfig, logger log.Logger, persistence *memory.InMemoryStatePersistence) {
	select {
	case <-ctx.Done():
		dump, err := serializer.NewStatePersistenceSerializer(persistence).Dump()
		if err != nil {
			logger.Error("failed to serialize state file", log.Error(err))
			return
		}

		if err := ioutil.WriteFile(getStateFilePath(config), dump, 0644); err != nil {
			logger.Error("failed not write state file", log.Error(err))
		}

		logger.Info("successfully serialized the state")
	}
}

func getStateFilePath(config config.NodeConfig) string {
	return filepath.Join(config.BlockStorageFileSystemDataDir(), "state")
}
