package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

type service struct {
	blockStorage         services.BlockStorage
	stateStorage         services.StateStorage
	processors           map[protocol.ProcessorType]services.Processor
	crosschainConnectors map[protocol.CrosschainConnectorType]services.CrosschainConnector
	reporting            instrumentation.BasicLogger

	contexts *executionContextProvider
}

func NewVirtualMachine(
	blockStorage services.BlockStorage,
	stateStorage services.StateStorage,
	processors map[protocol.ProcessorType]services.Processor,
	crosschainConnectors map[protocol.CrosschainConnectorType]services.CrosschainConnector,
	reporting instrumentation.BasicLogger,
) services.VirtualMachine {

	s := &service{
		blockStorage:         blockStorage,
		processors:           processors,
		crosschainConnectors: crosschainConnectors,
		stateStorage:         stateStorage,
		reporting:            reporting.For(instrumentation.Service("virtual-machine")),

		contexts: newExecutionContextProvider(),
	}

	for _, processor := range processors {
		processor.RegisterContractSdkCallHandler(s)
	}

	return s
}

func (s *service) ProcessTransactionSet(input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	previousBlockHeight := input.BlockHeight - 1 // our contracts rely on this block's state for execution
	receipts, stateDiffs := s.processTransactionSet(previousBlockHeight, input.SignedTransactions)
	s.reporting.Info("processed transaction set", instrumentation.BlockHeight(previousBlockHeight), instrumentation.Int("num-receipts", len(receipts)), instrumentation.Int("num-contract-state-diffs", len(stateDiffs)))

	return &services.ProcessTransactionSetOutput{
		TransactionReceipts: receipts,
		ContractStateDiffs:  stateDiffs,
	}, nil
}

func (s *service) RunLocalMethod(input *services.RunLocalMethodInput) (*services.RunLocalMethodOutput, error) {
	blockHeight, blockTimestamp, err := s.getRecentBlockHeight()
	if err != nil {
		return &services.RunLocalMethodOutput{
			CallResult:              protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
			OutputArguments:         []*protocol.MethodArgument{},
			ReferenceBlockHeight:    blockHeight,
			ReferenceBlockTimestamp: blockTimestamp,
		}, err
	}

	callResult, outputArgs, err := s.runLocalMethod(blockHeight, input.Transaction)
	// TODO: when we change the protos for RunLocalMethodOutput make the output args easily stringable for logging
	s.reporting.Info("ran local method", instrumentation.BlockHeight(blockHeight), instrumentation.Stringable("result", callResult), instrumentation.Error(err))

	return &services.RunLocalMethodOutput{
		CallResult:              callResult,
		OutputArguments:         outputArgs,
		ReferenceBlockHeight:    blockHeight,
		ReferenceBlockTimestamp: blockTimestamp,
	}, err
}

func (s *service) TransactionSetPreOrder(input *services.TransactionSetPreOrderInput) (*services.TransactionSetPreOrderOutput, error) {
	statuses := make([]protocol.TransactionStatus, len(input.SignedTransactions))

	// check subscription
	previousBlockHeight := input.BlockHeight - 1 // our contracts rely on this block's state for execution
	err := s.callGlobalPreOrderContract(previousBlockHeight)
	if err != nil {
		for i := 0; i < len(statuses); i++ {
			statuses[i] = protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER
		}
	} else {
		// check signatures
		err = s.verifyTransactionSignatures(input.SignedTransactions, statuses)
	}

	s.reporting.Info("performed pre order checks", instrumentation.Error(err), instrumentation.BlockHeight(previousBlockHeight), instrumentation.Int("num-statuses", len(statuses)))

	return &services.TransactionSetPreOrderOutput{
		PreOrderResults: statuses,
	}, err
}

func (s *service) HandleSdkCall(input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	var output []*protocol.MethodArgument
	var err error

	executionContext := s.contexts.loadExecutionContext(input.ContextId)
	if executionContext == nil {
		return nil, errors.Errorf("invalid execution context %s", input.ContextId)
	}

	switch input.ContractName {
	case native.SDK_STATE_CONTRACT_NAME:
		output, err = s.handleSdkStateCall(executionContext, input.MethodName, input.InputArguments)
	case native.SDK_SERVICE_CONTRACT_NAME:
		output, err = s.handleSdkServiceCall(executionContext, input.MethodName, input.InputArguments)
	default:
		return nil, errors.Errorf("unknown SDK call type: %s", input.ContractName)
	}

	if err != nil {
		return nil, err
	}

	return &handlers.HandleSdkCallOutput{
		OutputArguments: output,
	}, nil
}
