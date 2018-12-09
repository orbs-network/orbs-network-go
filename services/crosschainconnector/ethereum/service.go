package ethereum

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"strings"
)

type service struct {
	connection adapter.EthereumConnection
	logger     log.BasicLogger
}

func NewEthereumCrosschainConnector(connection adapter.EthereumConnection, logger log.BasicLogger) services.CrosschainConnector {
	s := &service{
		connection: connection,
		logger:     logger,
	}
	return s
}

func (s *service) EthereumCallContract(ctx context.Context, input *services.EthereumCallContractInput) (*services.EthereumCallContractOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("calling contract", log.String("contract-address", input.EthereumContractAddress), log.String("function", input.EthereumFunctionName))

	contractAddress, err := hexutil.Decode(input.EthereumContractAddress)
	if err != nil {
		return nil, err
	}

	// TODO(v1): use input.ReferenceTimestamp to find the reference block number (last param)
	output, err := s.connection.CallContract(ctx, contractAddress, input.EthereumAbiPackedInputArguments, nil)
	if err != nil {
		return nil, err
	}

	return &services.EthereumCallContractOutput{
		EthereumAbiPackedOutput: output,
	}, nil
}

func (s *service) EthereumGetTransactionLogs(ctx context.Context, input *services.EthereumGetTransactionLogsInput) (*services.EthereumGetTransactionLogsOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("getting transaction logs", log.String("contract-address", input.EthereumContractAddress), log.String("event", input.EthereumEventName), log.Stringable("txhash", input.EthereumTxhash))

	parsedABI, err := abi.JSON(strings.NewReader(input.EthereumJsonAbi))
	if err != nil {
		return nil, err
	}

	eventABI, found := parsedABI.Events[input.EthereumEventName]
	if !found {
		return nil, errors.Errorf("event with name '%s' not found in given ABI", input.EthereumEventName)
	}

	// TODO(v1): use input.ReferenceTimestamp to reduce non-determinism here (ask OdedW how)
	logs, err := s.connection.GetTransactionLogs(ctx, input.EthereumTxhash, eventABI.Id().Bytes())

	if err != nil {
		return nil, errors.Wrapf(err, "failed getting logs for Ethereum txhash %s of contract %s", input.EthereumTxhash, input.EthereumContractAddress)
	}

	// TODO(https://github.com/orbs-network/orbs-network-go/issues/597): support multiple logs
	if len(logs) != 1 {
		return nil, errors.Errorf("expected exactly one log entry for txhash %s of contract %s but got %d", input.EthereumTxhash, input.EthereumContractAddress, len(logs))
	}

	output, err := repackEventABIWithTopics(eventABI, logs[0])
	if err != nil {
		return nil, err
	}

	return &services.EthereumGetTransactionLogsOutput{
		EthereumAbiPackedOutput: output,
	}, nil
}
