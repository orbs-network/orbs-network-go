package ethereum

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
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
	// TODO: use input.ReferenceTimestamp to find the reference block number
	s.logger.Info("calling contract at", log.String("address", input.EthereumContractAddress))
	address, err := hexutil.Decode(input.EthereumContractAddress)
	if err != nil {
		return nil, err
	}
	contractAddress := common.BytesToAddress(address)

	// we do not support pending calls, opts is always empty
	opts := new(bind.CallOpts)
	client, err := s.connection.GetClient()
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{From: opts.From, To: &contractAddress, Data: input.EthereumPackedInputArguments}
	output, err := client.CallContract(ctx, msg, nil) // TODO: replace the last param with the calculated block number
	if err == nil && len(output) == 0 {
		// Make sure we have a contract to operate on, and bail out otherwise.
		if code, err := client.CodeAt(ctx, contractAddress, nil); err != nil {
			return nil, err
		} else if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}

	return &services.EthereumCallContractOutput{EthereumPackedOutput: output}, nil
}

func (s *service) EthereumGetTransactionLogs(ctx context.Context, input *services.EthereumGetTransactionLogsInput) (*services.EthereumGetTransactionLogsOutput, error) {
	panic("Not implemented")
}
