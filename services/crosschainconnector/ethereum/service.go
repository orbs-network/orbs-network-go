package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type service struct {
}

func NewEthereumCrosschainConnector() services.CrosschainConnector {
	return &service{}
}

func (s *service) EthereumCallContract(ctx context.Context, input *services.EthereumCallContractInput) (*services.EthereumCallContractOutput, error) {
	panic("Not implemented")
}
