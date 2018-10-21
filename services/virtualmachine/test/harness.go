package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"os"
)

type harness struct {
	blockStorage         *services.MockBlockStorage
	stateStorage         *services.MockStateStorage
	processors           map[protocol.ProcessorType]*services.MockProcessor
	crosschainConnectors map[protocol.CrosschainConnectorType]*services.MockCrosschainConnector
	reporting            log.BasicLogger
	service              services.VirtualMachine
}

func newHarness() *harness {
	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

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

	service := virtualmachine.NewVirtualMachine(
		stateStorage,
		processorsForService,
		crosschainConnectorsForService,
		log,
	)

	return &harness{
		blockStorage:         blockStorage,
		stateStorage:         stateStorage,
		processors:           processors,
		crosschainConnectors: crosschainConnectors,
		reporting:            log,
		service:              service,
	}
}

func (h *harness) handleSdkCall(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName primitives.ContractName, methodName primitives.MethodName, args ...interface{}) ([]*protocol.MethodArgument, error) {
	output, err := h.service.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:       executionContextId,
		OperationName:   contractName,
		MethodName:      methodName,
		InputArguments:  builders.MethodArguments(args...),
		PermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
	})
	if err != nil {
		return nil, err
	}
	return output.OutputArguments, nil
}

func (h *harness) handleSdkCallWithSystemPermissions(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName primitives.ContractName, methodName primitives.MethodName, args ...interface{}) ([]*protocol.MethodArgument, error) {
	output, err := h.service.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:       executionContextId,
		OperationName:   contractName,
		MethodName:      methodName,
		InputArguments:  builders.MethodArguments(args...),
		PermissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	})
	if err != nil {
		return nil, err
	}
	return output.OutputArguments, nil
}

func (h *harness) runLocalMethod(ctx context.Context, contractName primitives.ContractName, methodName primitives.MethodName) (protocol.ExecutionResult, []byte, primitives.BlockHeight, error) {
	output, err := h.service.RunLocalMethod(ctx, &services.RunLocalMethodInput{
		BlockHeight: 0,
		Transaction: (&protocol.TransactionBuilder{
			Signer:             nil,
			ContractName:       contractName,
			MethodName:         methodName,
			InputArgumentArray: []byte{},
		}).Build(),
	})
	return output.CallResult, output.OutputArgumentArray, output.ReferenceBlockHeight, err
}

type keyValuePair struct {
	key   primitives.Ripmd160Sha256
	value []byte
}

type contractAndMethod struct {
	contractName primitives.ContractName
	methodName   primitives.MethodName
}

func (h *harness) processTransactionSet(ctx context.Context, contractAndMethods []*contractAndMethod) ([]protocol.ExecutionResult, [][]byte, map[primitives.ContractName][]*keyValuePair) {
	resultKeyValuePairsPerContract := make(map[primitives.ContractName][]*keyValuePair)

	transactions := []*protocol.SignedTransaction{}
	for _, contractAndMethod := range contractAndMethods {
		resultKeyValuePairsPerContract[contractAndMethod.contractName] = []*keyValuePair{}
		tx := builders.Transaction().WithMethod(contractAndMethod.contractName, contractAndMethod.methodName).Build()
		transactions = append(transactions, tx)
	}

	output, _ := h.service.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		BlockHeight:        12,
		SignedTransactions: transactions,
	})

	results := []protocol.ExecutionResult{}
	outputArgsOfAllTransactions := [][]byte{}
	for _, transactionReceipt := range output.TransactionReceipts {
		result := transactionReceipt.ExecutionResult()
		results = append(results, result)
		outputArgs := transactionReceipt.OutputArgumentArray()
		outputArgsOfAllTransactions = append(outputArgsOfAllTransactions, outputArgs)
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

	return results, outputArgsOfAllTransactions, resultKeyValuePairsPerContract
}

func (h *harness) transactionSetPreOrder(ctx context.Context, signedTransactions []*protocol.SignedTransaction) ([]protocol.TransactionStatus, error) {
	output, err := h.service.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		BlockHeight:        12,
		SignedTransactions: signedTransactions,
	})
	return output.PreOrderResults, err
}
