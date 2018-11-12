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
	client    bind.ContractBackend
	config    ethereumConnectorConfig
}

func NewEthereumCrosschainConnector(ctx context.Context, config ethereumConnectorConfig, connector adapter.EthereumConnection, logger log.BasicLogger) services.CrosschainConnector {
	s := &service{
		connector: connector,
		client:    nil,
		logger:    logger,
		config:    config,
	}

	return s
}

func (s *service) Connect() error {
	client, err := s.connector.Dial(s.config.EthereumEndpoint())
	if err != nil {
		return err
	}
	s.client = client
	return nil
}

func (s *service) EthereumCallContract(ctx context.Context, input *services.EthereumCallContractInput) (*services.EthereumCallContractOutput, error) {
	panic("Not implemented")
	opts := new(bind.CallOpts)
	address, err := hexutil.Decode(input.EthereumContractAddress)
	if err != nil {
		return nil, err
	}

	contractAddress := common.BytesToAddress(address)

	var (
		msg    = ethereum.CallMsg{From: opts.From, To: &contractAddress, Data: input.EthereumPackedInputArguments}
		code   []byte
		output []byte
	)

	// we do not support pending calls
	output, err = s.client.CallContract(ctx, msg, nil)
	if err == nil && len(output) == 0 {
		// Make sure we have a contract to operate on, and bail out otherwise.
		if code, err = s.client.CodeAt(ctx, contractAddress, nil); err != nil {
			return nil, err
		} else if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}

	return &services.EthereumCallContractOutput{EthereumPackedOutput: output}, nil
}
