package ethereum

import (
	"context"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

type service struct {
	connection adapter.EthereumConnection
	logger     log.BasicLogger
}

func NewEthereumCrosschainConnector(ctx context.Context, // TODO: why don't we use context here?
	connection adapter.EthereumConnection,
	logger log.BasicLogger) services.CrosschainConnector {
	s := &service{
		connection: connection,
		logger:     logger,
	}

	return s
}

func (s *service) EthereumCallContract(ctx context.Context, input *services.EthereumCallContractInput) (*services.EthereumCallContractOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	// TODO: use input.ReferenceTimestamp to find the reference block number
	logger.Info("calling contract at", log.String("address", input.EthereumContractAddress))
	address, err := hexutil.Decode(input.EthereumContractAddress)
	if err != nil {
		return nil, err
	}

	output, err := s.connection.CallContract(ctx, address, input.EthereumPackedInputArguments, nil) // TODO: replace the last param with the calculated block number
	if err != nil {
		return nil, err
	}

	return &services.EthereumCallContractOutput{EthereumPackedOutput: output}, nil
}

func (s *service) EthereumGetTransactionLogs(ctx context.Context, input *services.EthereumGetTransactionLogsInput) (*services.EthereumGetTransactionLogsOutput, error) {
	logs, err := s.connection.GetLogs(ctx, input.EthereumTxhash, []byte(input.EthereumContractAddress), []byte(input.EventSignature))
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting logs for Ethereum txhash %s of contract %s", input.EthereumTxhash, input.EthereumContractAddress)
	}

	if len(logs) != 1 {
		return nil, errors.Errorf("expected exactly one log entry for txhash %s of contract %s but got %d", input.EthereumTxhash, input.EthereumContractAddress, len(logs))
	}

	out := &services.EthereumGetTransactionLogsOutput{
		EthereumPackedEventData: logs[0].Data,
		EthereumPackedEventTopics: logs[0].PackedTopics,
	}

	return out, nil
}
