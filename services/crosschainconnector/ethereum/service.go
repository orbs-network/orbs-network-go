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

var LogTag = log.Service("crosschain-connector")

type service struct {
	connection       adapter.EthereumConnection
	logger           log.BasicLogger
	timestampFetcher adapter.TimestampFetcher
}

func NewEthereumCrosschainConnector(connection adapter.EthereumConnection, parent log.BasicLogger) services.CrosschainConnector {
	logger := parent.WithTags(LogTag)
	s := &service{
		connection:       connection,
		timestampFetcher: adapter.NewTimestampFetcher(adapter.NewBlockTimestampFetcher(connection), logger),
		logger:           logger,
	}
	return s
}

func (s *service) EthereumCallContract(ctx context.Context, input *services.EthereumCallContractInput) (*services.EthereumCallContractOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	referenceBlockNumber, err := s.timestampFetcher.GetBlockByTimestamp(ctx, input.ReferenceTimestamp)
	if err != nil {
		return nil, err
	}
	if referenceBlockNumber != nil {
		logger.Info("calling contract from ethereum",
			log.String("address", input.EthereumContractAddress),
			log.Int64("reference-block", referenceBlockNumber.Int64()))
	}
	address, err := hexutil.Decode(input.EthereumContractAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode the contract address %s", input.EthereumContractAddress)
	}

	output, err := s.connection.CallContract(ctx, address, input.EthereumAbiPackedInputArguments, referenceBlockNumber)
	if err != nil {
		return nil, errors.Wrap(err, "ethereum call failed")
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
