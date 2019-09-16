package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
)

type NopEthereumAdapter struct {
}

func (n NopEthereumAdapter) CallContract(ctx context.Context, contractAddress []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error) {
	return nil, errors.Errorf("I'm the NOP Ethereum Connector")
}

func (n NopEthereumAdapter) GetTransactionLogs(ctx context.Context, txHash primitives.Uint256, eventSignature []byte) ([]*TransactionLog, error) {
	return nil, errors.Errorf("I'm the NOP Ethereum Connector")
}

func (n NopEthereumAdapter) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, errors.Errorf("I'm the NOP Ethereum Connector")
}
