// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"time"
)

var LogTag = log.Service("virtual-machine")

type ManagementConfig interface {
	CommitteeGracePeriod() time.Duration
}

type service struct {
	stateStorage         services.StateStorage
	processors           map[protocol.ProcessorType]services.Processor
	crosschainConnectors map[protocol.CrosschainConnectorType]services.CrosschainConnector
	management           services.Management
	cfg                  ManagementConfig
	logger               log.Logger

	contexts *executionContextProvider
}

func NewVirtualMachine(stateStorage services.StateStorage, processors map[protocol.ProcessorType]services.Processor, crosschainConnectors map[protocol.CrosschainConnectorType]services.CrosschainConnector, management services.Management, cfg ManagementConfig, logger log.Logger) services.VirtualMachine {
	s := &service{
		processors:           processors,
		crosschainConnectors: crosschainConnectors,
		stateStorage:         stateStorage,
		management:           management,
		cfg:                  cfg,
		logger:               logger.WithTags(LogTag),

		contexts: newExecutionContextProvider(),
	}

	for _, processor := range processors {
		processor.RegisterContractSdkCallHandler(s)
	}

	return s
}

func (s *service) ProcessQuery(ctx context.Context, input *services.ProcessQueryInput) (*services.ProcessQueryOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	committedBlockHeight, committedBlockTimestamp, committeeReferenceTime, committedPrevReferenceTime, committedBlockProposerAddress, err := s.getRecentCommittedBlockInfo(ctx)
	if err != nil {
		return &services.ProcessQueryOutput{
			CallResult:              protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
			OutputArgumentArray:     protocol.ArgumentsArrayEmpty().Raw(),
			ReferenceBlockHeight:    committedBlockHeight,
			ReferenceBlockTimestamp: committedBlockTimestamp,
		}, err
	}

	if input.BlockHeight != 0 {
		err := errors.New("Run local method with specific block height is not yet supported")
		return &services.ProcessQueryOutput{
			CallResult:              protocol.EXECUTION_RESULT_ERROR_INPUT,
			OutputArgumentArray:     protocol.ArgumentsArrayEmpty().Raw(),
			ReferenceBlockHeight:    committedBlockHeight,
			ReferenceBlockTimestamp: committedBlockTimestamp,
		}, err
	}

	logger.Info("running local method", log.Stringable("contract", input.SignedQuery.Query().ContractName()), log.Stringable("method", input.SignedQuery.Query().MethodName()), logfields.BlockHeight(committedBlockHeight))
	callResult, outputArgs, outputEvents, err := s.runMethod(ctx, committedBlockHeight, committedBlockHeight, committedBlockTimestamp, committedBlockProposerAddress, committeeReferenceTime, committedPrevReferenceTime, input.SignedQuery.Query(), protocol.ACCESS_SCOPE_READ_ONLY, nil)
	if outputArgs == nil {
		outputArgs = protocol.ArgumentsArrayEmpty()
	}
	if outputEvents == nil {
		outputEvents = (&protocol.EventsArrayBuilder{}).Build()
	}

	return &services.ProcessQueryOutput{
		CallResult:              callResult,
		OutputEventsArray:       outputEvents.RawEventsArray(),
		OutputArgumentArray:     outputArgs.RawArgumentsArray(),
		ReferenceBlockHeight:    committedBlockHeight,
		ReferenceBlockTimestamp: committedBlockTimestamp,
	}, err
}

func (s *service) ProcessTransactionSet(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	logger.Info("processing transaction set", log.Int("num-transactions", len(input.SignedTransactions)), logfields.BlockHeight(input.CurrentBlockHeight))
	receipts, stateDiffs := s.processTransactionSet(ctx, input.CurrentBlockHeight, input.CurrentBlockTimestamp, input.BlockProposerAddress, input.CurrentBlockReferenceTime, input.PrevBlockReferenceTime, input.SignedTransactions)

	return &services.ProcessTransactionSetOutput{
		TransactionReceipts: receipts,
		ContractStateDiffs:  stateDiffs,
	}, nil
}

func (s *service) TransactionSetPreOrder(ctx context.Context, input *services.TransactionSetPreOrderInput) (*services.TransactionSetPreOrderOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	// all statuses start as protocol.TRANSACTION_STATUS_RESERVED (zero)
	statuses := make([]protocol.TransactionStatus, len(input.SignedTransactions))

	// Check Subscription and Committee during pre-order execution to allow empty (rejected status) yet "valid" blocks when either of them fail
	isSubscriptionActive := s.verifySubscription(ctx, input.CurrentBlockReferenceTime)
	isCommitteeActive := s.verifyCommitteeStatus(input.CurrentBlockTimestamp, input.CurrentBlockReferenceTime)
	if !isSubscriptionActive || !isCommitteeActive {
		for i := 0; i < len(input.SignedTransactions); i++ {
			statuses[i] = protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER
		}
	} else {
		// check signatures
		s.verifyTransactionSignatures(input.SignedTransactions, statuses)
	}

	if !isSubscriptionActive {
		logger.Info("performed pre order checks", log.Error(errors.New("Subscription Expired")), logfields.BlockHeight(input.CurrentBlockHeight), log.Int("num-statuses", len(statuses)))
	} else if !isCommitteeActive {
		logger.Error("performed pre order checks", log.Error(errors.New("Network has lost live connection to management")), logfields.BlockHeight(input.CurrentBlockHeight), log.Int("num-statuses", len(statuses)))
	} else {
		logger.Info("performed pre order checks", logfields.BlockHeight(input.CurrentBlockHeight), log.Int("num-statuses", len(statuses)))
	}

	return &services.TransactionSetPreOrderOutput{
		PreOrderResults: statuses,
	}, nil
}

func (s *service) CallSystemContract(ctx context.Context, input *services.CallSystemContractInput) (*services.CallSystemContractOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	logger.Info("calling system contract", log.Stringable("contract", input.ContractName), log.Stringable("method", input.MethodName), logfields.BlockHeight(input.BlockHeight))
	callResult, outputArgs, err := s.callSystemContract(ctx, input.BlockHeight, input.BlockTimestamp, input.CurrentBlockReferenceTime, input.PrevBlockReferenceTime, input.ContractName, input.MethodName, input.InputArgumentArray)
	if outputArgs == nil {
		outputArgs = protocol.ArgumentsArrayEmpty()
	}

	return &services.CallSystemContractOutput{
		OutputArgumentArray: outputArgs,
		CallResult:          callResult,
	}, err
}

func (s *service) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	var output []*protocol.Argument
	var err error

	executionContext := s.contexts.loadExecutionContext(input.ContextId)
	if executionContext == nil {
		return nil, errors.Errorf("invalid execution context %s", input.ContextId)
	}

	switch input.OperationName {
	case sdk.SDK_OPERATION_NAME_STATE:
		output, err = s.handleSdkStateCall(ctx, executionContext, input.MethodName, input.InputArguments, input.PermissionScope)
	case sdk.SDK_OPERATION_NAME_SERVICE:
		output, err = s.handleSdkServiceCall(ctx, executionContext, input.MethodName, input.InputArguments, input.PermissionScope)
	case sdk.SDK_OPERATION_NAME_EVENTS:
		output, err = s.handleSdkEventsCall(ctx, executionContext, input.MethodName, input.InputArguments, input.PermissionScope)
	case sdk.SDK_OPERATION_NAME_ETHEREUM:
		output, err = s.handleSdkEthereumCall(ctx, executionContext, input.MethodName, input.InputArguments, input.PermissionScope)
	case sdk.SDK_OPERATION_NAME_ADDRESS:
		output, err = s.handleSdkAddressCall(ctx, executionContext, input.MethodName, input.InputArguments, input.PermissionScope)
	case sdk.SDK_OPERATION_NAME_ENV:
		output, err = s.handleSdkEnvCall(ctx, executionContext, input.MethodName, input.InputArguments, input.PermissionScope)
	default:
		return nil, errors.Errorf("unknown SDK call operation: %s", input.OperationName)
	}

	if err != nil {
		return nil, err
	}

	return &handlers.HandleSdkCallOutput{
		OutputArguments: output,
	}, nil
}
