package ethereum

import (
	"context"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-spec/types/go/services"
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
