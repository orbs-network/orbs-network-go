// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"time"
)

type harness struct {
	blockStorage         *services.MockBlockStorage
	stateStorage         *services.MockStateStorage
	processors           map[protocol.ProcessorType]*services.MockProcessor
	crosschainConnectors map[protocol.CrosschainConnectorType]*services.MockCrosschainConnector
	managementProvider   *managementProvider
	logger               log.Logger
	service              services.VirtualMachine
}

func newHarness(logger log.Logger) *harness {
	blockStorage := &services.MockBlockStorage{}
	stateStorage := &services.MockStateStorage{}

	processors := make(map[protocol.ProcessorType]*services.MockProcessor)
	processors[protocol.PROCESSOR_TYPE_NATIVE] = &services.MockProcessor{}
	processors[protocol.PROCESSOR_TYPE_NATIVE].When("RegisterContractSdkCallHandler", mock.Any).Return().Times(1)

	crosschainConnectors := make(map[protocol.CrosschainConnectorType]*services.MockCrosschainConnector)
	crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM] = &services.MockCrosschainConnector{}

	processorsForService := make(map[protocol.ProcessorType]services.Processor)
	for key, value := range processors {
		processorsForService[key] = value
	}

	crosschainConnectorsForService := make(map[protocol.CrosschainConnectorType]services.CrosschainConnector)
	for key, value := range crosschainConnectors {
		crosschainConnectorsForService[key] = value
	}

	managementProvider := NewTestManagementProvider(4)
	service := virtualmachine.NewVirtualMachine(stateStorage, processorsForService, crosschainConnectorsForService, managementProvider, managementProvider, logger)

	return &harness{
		blockStorage:         blockStorage,
		stateStorage:         stateStorage,
		processors:           processors,
		crosschainConnectors: crosschainConnectors,
		managementProvider:   managementProvider,
		logger:               logger,
		service:              service,
	}
}

func (h *harness) handleSdkCall(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName primitives.ContractName, methodName primitives.MethodName, args ...interface{}) ([]*protocol.Argument, error) {
	inputArgs, err := protocol.ArgumentsFromNatives(args)
	if err != nil {
		return nil, err
	}
	output, err := h.service.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:       executionContextId,
		OperationName:   contractName,
		MethodName:      methodName,
		InputArguments:  inputArgs,
		PermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
	})
	if err != nil {
		return nil, err
	}
	return output.OutputArguments, nil
}

func (h *harness) handleSdkCallWithSystemPermissions(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName primitives.ContractName, methodName primitives.MethodName, args ...interface{}) ([]*protocol.Argument, error) {
	inputArgs, err := protocol.ArgumentsFromNatives(args)
	if err != nil {
		return nil, err
	}
	output, err := h.service.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:       executionContextId,
		OperationName:   contractName,
		MethodName:      methodName,
		InputArguments:  inputArgs,
		PermissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	})
	if err != nil {
		return nil, err
	}
	return output.OutputArguments, nil
}

func (h *harness) processQuery(ctx context.Context, contractName primitives.ContractName, methodName primitives.MethodName) (protocol.ExecutionResult, []byte, primitives.BlockHeight, []byte, error) {
	output, err := h.service.ProcessQuery(ctx, &services.ProcessQueryInput{
		BlockHeight: 0, // recent
		SignedQuery: (&protocol.SignedQueryBuilder{
			Query: &protocol.QueryBuilder{
				Signer:             nil,
				ContractName:       contractName,
				MethodName:         methodName,
				InputArgumentArray: []byte{},
			},
		}).Build(),
	})
	return output.CallResult, output.OutputArgumentArray, output.ReferenceBlockHeight, output.OutputEventsArray, err
}

func (h *harness) callSystemContract(ctx context.Context, blockHeight primitives.BlockHeight, contractName primitives.ContractName, methodName primitives.MethodName, args ...interface{}) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
	inputArgArray, err := protocol.ArgumentArrayFromNatives(args)
	if err != nil {
		return protocol.EXECUTION_RESULT_ERROR_INPUT, nil, err
	}
	output, err := h.service.CallSystemContract(ctx, &services.CallSystemContractInput{
		BlockHeight:        blockHeight,
		BlockTimestamp:     0,
		ContractName:       contractName,
		MethodName:         methodName,
		InputArgumentArray: inputArgArray,
	})
	return output.CallResult, output.OutputArgumentArray, err
}

type keyValuePair struct {
	key   []byte
	value []byte
}

type contractAndMethod struct {
	contractName primitives.ContractName
	methodName   primitives.MethodName
}

func (h *harness) processTransactionSet(ctx context.Context, contractAndMethods []*contractAndMethod, additionalExpectedStateDiffContracts ...primitives.ContractName) ([]protocol.ExecutionResult, [][]byte, map[primitives.ContractName][]*keyValuePair, [][]byte) {
	return h.processTransactionSetWithBlockInfo(ctx, 12, 0x777, hash.Make32BytesWithFirstByte(5), contractAndMethods, additionalExpectedStateDiffContracts...)
}

func (h *harness) processTransactionSetWithBlockInfo(ctx context.Context, currentBlockHeight primitives.BlockHeight, currentBlockTimestamp primitives.TimestampNano, currentBlockProposer primitives.NodeAddress, contractAndMethods []*contractAndMethod, additionalExpectedStateDiffContracts ...primitives.ContractName) ([]protocol.ExecutionResult, [][]byte, map[primitives.ContractName][]*keyValuePair, [][]byte) {
	resultKeyValuePairsPerContract := make(map[primitives.ContractName][]*keyValuePair)

	transactions := []*protocol.SignedTransaction{}
	for _, contractAndMethod := range contractAndMethods {
		resultKeyValuePairsPerContract[contractAndMethod.contractName] = []*keyValuePair{}
		tx := builders.Transaction().WithMethod(contractAndMethod.contractName, contractAndMethod.methodName).Build()
		transactions = append(transactions, tx)
	}
	for _, contractName := range additionalExpectedStateDiffContracts {
		resultKeyValuePairsPerContract[contractName] = []*keyValuePair{}
	}

	output, _ := h.service.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		SignedTransactions:    transactions,
		CurrentBlockHeight:    currentBlockHeight,
		CurrentBlockTimestamp: currentBlockTimestamp,
		BlockProposerAddress:  currentBlockProposer,
	})

	results := []protocol.ExecutionResult{}
	outputArgsOfAllTransactions := [][]byte{}
	outputEventsOfAllTransactions := [][]byte{}
	for _, transactionReceipt := range output.TransactionReceipts {
		result := transactionReceipt.ExecutionResult()
		results = append(results, result)
		outputArgs := transactionReceipt.OutputArgumentArray()
		outputArgsOfAllTransactions = append(outputArgsOfAllTransactions, outputArgs)
		outputEventsOfAllTransactions = append(outputEventsOfAllTransactions, transactionReceipt.OutputEventsArray())
	}

	for _, contractStateDiffs := range output.ContractStateDiffs {
		contractName := contractStateDiffs.ContractName()
		if _, found := resultKeyValuePairsPerContract[contractName]; !found {
			panic(fmt.Sprintf("unexpected contract %s", contractStateDiffs.ContractName()))
		}
		for i := contractStateDiffs.StateDiffsIterator(); i.HasNext(); {
			sd := i.NextStateDiffs()
			resultKeyValuePairsPerContract[contractName] = append(resultKeyValuePairsPerContract[contractName], &keyValuePair{sd.Key(), sd.Value()})
		}
	}

	return results, outputArgsOfAllTransactions, resultKeyValuePairsPerContract, outputEventsOfAllTransactions
}

func (h *harness) transactionSetPreOrder(ctx context.Context, signedTransactions []*protocol.SignedTransaction) ([]protocol.TransactionStatus, error) {
	output, err := h.service.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		SignedTransactions:    signedTransactions,
		CurrentBlockHeight:    12,
		CurrentBlockTimestamp: 0x777,
	})
	return output.PreOrderResults, err
}

// TODO POSV2 REFTIME - replace with mgmt service mock
type managementProvider struct {
	nodes []primitives.NodeAddress
	isSubscription bool
	liveTime time.Duration
}

func NewTestManagementProvider(numOfNodes int) *managementProvider {
	return  &managementProvider{nodes:testKeys.NodeAddressesForTests()[:numOfNodes], isSubscription:true, liveTime: 1*time.Minute}
}

func (mp *managementProvider) GetCommittee(ctx context.Context, referenceTime primitives.TimestampSeconds) []primitives.NodeAddress {
	return mp.nodes
}

func (mp *managementProvider) GetSubscriptionStatus(ctx context.Context, reference primitives.TimestampSeconds) bool {
	return mp.isSubscription
}

func (mp *managementProvider) setSubscriptionStatus(isSub bool) {
	mp.isSubscription = isSub
}

func (mp *managementProvider) ManagementNetworkLivenessTimeout() time.Duration {
	return mp.liveTime
}
