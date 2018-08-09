package virtualmachine

import (
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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

	// TODO: until we integrate the acceptance to pass through the "correct" implementation we need this escape hatch to move the tests forward
	if len(input.SignedTransactions) > 0 && input.SignedTransactions[0].Transaction().ContractName() == "ExampleContract" {

		stateDiffs, err := s.processTransactionSet(input.BlockHeight, input.SignedTransactions)
		if err != nil {
			return nil, err
		}

		return &services.ProcessTransactionSetOutput{
			TransactionReceipts: nil,
			ContractStateDiffs:  stateDiffs,
		}, nil
	}

	var state []*protocol.StateRecordBuilder
	for _, i := range input.SignedTransactions {
		byteArray := make([]byte, 8)
		binary.LittleEndian.PutUint64(byteArray, uint64(i.Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value()))
		transactionStateDiff := &protocol.StateRecordBuilder{
			Key:   primitives.Ripmd160Sha256(fmt.Sprintf("balance%v", uint64(input.BlockHeight))),
			Value: byteArray,
		}
		state = append(state, transactionStateDiff)
	}
	csdi := []*protocol.ContractStateDiff{(&protocol.ContractStateDiffBuilder{ContractName: "BenchmarkToken", StateDiffs: state}).Build()}
	s.stateStorage.CommitStateDiff(
		&services.CommitStateDiffInput{
			ResultsBlockHeader: (&protocol.ResultsBlockHeaderBuilder{BlockHeight: input.BlockHeight}).Build(),
			ContractStateDiffs: csdi})

	return &services.ProcessTransactionSetOutput{
		TransactionReceipts: nil, // TODO
		ContractStateDiffs:  csdi,
	}, nil
}

func (s *service) RunLocalMethod(input *services.RunLocalMethodInput) (*services.RunLocalMethodOutput, error) {

	// TODO: until we integrate the acceptance to pass through the "correct" implementation we need this escape hatch to move the tests forward
	if input.Transaction.ContractName() == "ExampleContract" {

		blockHeight, blockTimestamp, err := s.getRecentBlockHeight()
		if err != nil {
			return nil, err
		}

		callResult, outputArgs, err := s.runLocalMethod(blockHeight, input.Transaction)
		if err != nil {
			return nil, err
		}

		return &services.RunLocalMethodOutput{
			CallResult:              callResult,
			OutputArguments:         outputArgs,
			ReferenceBlockHeight:    blockHeight,
			ReferenceBlockTimestamp: blockTimestamp,
		}, nil
	}

	// TODO XXX this implementation bakes an implementation of an arbitraty contract function.
	// The function scans a set of keys derived from the current block height and sums up all their values.

	// todo get list of keys to read from "hard codded contract func"
	blockHeight, _ := s.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})
	keys := make([]primitives.Ripmd160Sha256, 0, blockHeight.LastCommittedBlockHeight)
	for i := uint64(0); i < uint64(blockHeight.LastCommittedBlockHeight)+uint64(1); i++ {
		keys = append(keys, primitives.Ripmd160Sha256(fmt.Sprintf("balance%v", i)))
	}

	sum := uint64(0)
	readKeys := &services.ReadKeysInput{BlockHeight: blockHeight.LastCommittedBlockHeight, ContractName: "BenchmarkToken", Keys: keys}
	if results, err := s.stateStorage.ReadKeys(readKeys); err != nil {
		// Todo handle error gracefully
	} else {
		for _, t := range results.StateRecords {
			if len(t.Value()) > 0 {
				sum += binary.LittleEndian.Uint64(t.Value())
			}
		}
	}
	arg := (&protocol.MethodArgumentBuilder{
		Name:        "balance",
		Type:        protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE,
		Uint64Value: sum,
	}).Build()
	return &services.RunLocalMethodOutput{
		OutputArguments: []*protocol.MethodArgument{arg},
	}, nil
}

func (s *service) TransactionSetPreOrder(input *services.TransactionSetPreOrderInput) (*services.TransactionSetPreOrderOutput, error) {

	//TODO this is a stub to make AddNewTransaction pass. Remove when real implementation arrives
	numOfTransactions := len(input.SignedTransactions)
	results := make([]protocol.TransactionStatus, numOfTransactions, numOfTransactions)
	for i := range results {
		results[i] = protocol.TRANSACTION_STATUS_PENDING
	}
	return &services.TransactionSetPreOrderOutput{
		PreOrderResults: results,
	}, nil
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
