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

type ethereumConnectorConfig interface {
	EthereumEndpoint() string
}

type service struct {
	connector adapter.EthereumConnection
	logger    log.BasicLogger
	config    ethereumConnectorConfig
}

func NewEthereumCrosschainConnector(ctx context.Context, // TODO: why don't we use context here?
	config ethereumConnectorConfig,
	connector adapter.EthereumConnection,
	logger log.BasicLogger) services.CrosschainConnector {
	s := &service{
		connector: connector,
		logger:    logger,
		config:    config,
	}

	return s
}

func (s *service) setupClient() error {
	if s.connector.GetClient() == nil {
		s.logger.Info("connecting to ethereum", log.String("endpoint", s.config.EthereumEndpoint()))
		if err := s.connector.Dial(s.config.EthereumEndpoint()); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) EthereumCallContract(ctx context.Context, input *services.EthereumCallContractInput) (*services.EthereumCallContractOutput, error) {
	if err := s.setupClient(); err != nil { // lazy setup - if the eth node is down it will retry setup on next call
		return nil, err
	}
	s.logger.Info("calling contract at", log.String("address", input.EthereumContractAddress))
	address, err := hexutil.Decode(input.EthereumContractAddress)
	if err != nil {
		return nil, err
	}
	contractAddress := common.BytesToAddress(address)

	// we do not support pending calls, opts is always empty
	opts := new(bind.CallOpts)
	msg := ethereum.CallMsg{From: opts.From, To: &contractAddress, Data: input.EthereumPackedInputArguments}
	output, err := s.connector.GetClient().CallContract(ctx, msg, nil)
	if err == nil && len(output) == 0 {
		// Make sure we have a contract to operate on, and bail out otherwise.
		if code, err := s.connector.GetClient().CodeAt(ctx, contractAddress, nil); err != nil {
			return nil, err
		} else if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}

	return &services.EthereumCallContractOutput{EthereumPackedOutput: output}, nil
}
