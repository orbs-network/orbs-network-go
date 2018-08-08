package virtualmachine

import (
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type service struct {
	blockStorage         services.BlockStorage
	stateStorage         services.StateStorage
	processors           map[protocol.ProcessorType]services.Processor
	crosschainConnectors map[protocol.CrosschainConnectorType]services.CrosschainConnector
	reporting            instrumentation.BasicLogger
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
	}

	for _, processor := range processors {
		processor.RegisterContractSdkCallHandler(s)
	}

	return s
}

func (s *service) ProcessTransactionSet(input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {

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

	// TODO XXX this implementation bakes an implementation of an arbitraty contract function.
	// The function scans a set of keys derived from the current block height and sums up all their values.

	// todo get list of keys to read from "hard codded contract func"
	blockHeight, _ := s.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})
	keys := make([]primitives.Ripmd160Sha256, 0, blockHeight.LastCommittedBlockHeight)
	for i := uint64(0); i < uint64(blockHeight.LastCommittedBlockHeight)+uint64(1); i++ {
		keys = append(keys, primitives.Ripmd160Sha256(fmt.Sprintf("balance%v", i)))
	}

	readKeys := &services.ReadKeysInput{ContractName: "BenchmarkToken", Keys: keys}
	results, _ := s.stateStorage.ReadKeys(readKeys)
	sum := uint64(0)
	for _, t := range results.StateRecords {
		if len(t.Value()) > 0 {
			sum += binary.LittleEndian.Uint64(t.Value())
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
	panic("Not implemented")
}
