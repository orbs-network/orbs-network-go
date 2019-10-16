package sdk

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/processor/native/testkit"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
)

type harness struct {
	vm         services.VirtualMachine
	repository *testkit.ManualRepository
}

func newVmHarness(logger log.Logger) *harness {
	registry := metric.NewRegistry()

	ssCfg := config.ForStateStorageTest(10, 5, 5000)
	ssPersistence := memory.NewStatePersistence(registry)
	stateStorage := statestorage.NewStateStorage(ssCfg, ssPersistence, nil, logger, registry)

	sdkCallHandler := &handlers.MockContractSdkCallHandler{}
	psCfg := config.ForNativeProcessorTests(42)
	repo := testkit.NewRepository()

	processorService := native.NewProcessorWithContractRepository(repo, psCfg, logger, registry)
	processorService.RegisterContractSdkCallHandler(sdkCallHandler)

	processorMap := map[protocol.ProcessorType]services.Processor{protocol.PROCESSOR_TYPE_NATIVE: processorService}
	crosschainConnectors := make(map[protocol.CrosschainConnectorType]services.CrosschainConnector)
	crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM] = &services.MockCrosschainConnector{}
	vm := virtualmachine.NewVirtualMachine(stateStorage, processorMap, crosschainConnectors, logger)

	return &harness{
		vm:         vm,
		repository: repo,
	}
}
