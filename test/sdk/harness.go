// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package sdk

import (
	"context"
	"errors"
	"fmt"
	"github.com/orbs-network/crypto-lib-go/crypto/hash"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/processor/native/testkit"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	stateAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"time"
)

const ContractName = "TestContract"

func generateDeployTx() *protocol.SignedTransaction {
	return builders.Transaction().WithMethod("_Deployments", "deployService").
		WithArgs(ContractName, uint32(protocol.PROCESSOR_TYPE_NATIVE), []byte("irrelevant data - contract is already registered")).
		Build()
}

type harness struct {
	vm         services.VirtualMachine
	repository *testkit.ManualRepository
}

func newVmHarness(logger log.Logger) *harness {
	registry := metric.NewRegistry()

	ssCfg := config.ForStateStorageTest(10, 5, 5000)
	ssPersistence := stateAdapter.NewStatePersistence(registry)
	stateStorage := statestorage.NewStateStorage(ssCfg, ssPersistence, nil, logger, registry)

	management := &services.MockManagement{}
	management.When("GetCommittee", mock.Any, mock.Any).Return(&services.GetCommitteeOutput{Members: testKeys.NodeAddressesForTests()[:5]}, nil)

	sdkCallHandler := &handlers.MockContractSdkCallHandler{}
	psCfg := config.ForNativeProcessorTests(42)
	repo := testkit.NewRepository()

	processorService := native.NewProcessorWithContractRepository(repo, psCfg, logger, registry)
	processorService.RegisterContractSdkCallHandler(sdkCallHandler)

	processorMap := map[protocol.ProcessorType]services.Processor{protocol.PROCESSOR_TYPE_NATIVE: processorService}
	crosschainConnectors := make(map[protocol.CrosschainConnectorType]services.CrosschainConnector)
	crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM] = &services.MockCrosschainConnector{}
	vm := virtualmachine.NewVirtualMachine(stateStorage, processorMap, crosschainConnectors, management, &vmCfg{}, logger)

	return &harness{
		vm:         vm,
		repository: repo,
	}
}

func (h *harness) processSuccessfully(ctx context.Context, txs ...*protocol.SignedTransaction) ([]*protocol.TransactionReceipt, error) {
	out, err := h.process(ctx, txs...)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(txs); i++ {
		executionResult := out.TransactionReceipts[i].ExecutionResult()
		if protocol.EXECUTION_RESULT_SUCCESS != out.TransactionReceipts[i].ExecutionResult() {
			return nil, errors.New(fmt.Sprintf("tx %d should succeed. execution res was %s", i, executionResult))
		}
	}

	return out.TransactionReceipts, nil
}

func (h *harness) process(ctx context.Context, txs ...*protocol.SignedTransaction) (*services.ProcessTransactionSetOutput, error) {
	return h.vm.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		CurrentBlockHeight:    1,
		CurrentBlockTimestamp: 66,
		SignedTransactions:    txs,
		BlockProposerAddress:  hash.Make32BytesWithFirstByte(5),
	})
}

type vmCfg struct{}

func (c *vmCfg) ManagementNetworkLivenessTimeout() time.Duration {
	return 10 * time.Minute
}
