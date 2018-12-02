package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"math/big"
)

type ethereumAdapterConfig interface {
	EthereumEndpoint() string
}

type EthereumConnection interface {
	CallContract(ctx context.Context, address []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error)
}

type connectorCommon struct {
	getContractCaller func() (EthereumCaller, error)
}

type EthereumCaller interface {
	bind.ContractBackend
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

func (c *connectorCommon) CallContract(ctx context.Context, address []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	contractAddress := common.BytesToAddress(address)

	// we do not support pending calls, opts is always empty
	opts := new(bind.CallOpts)

	msg := ethereum.CallMsg{From: opts.From, To: &contractAddress, Data: packedInput}
	output, err := client.CallContract(ctx, msg, blockNumber)
	if err == nil && len(output) == 0 {
		// Make sure we have a contract to operate on, and bail out otherwise.
		if code, err := client.CodeAt(ctx, contractAddress, blockNumber); err != nil {
			return nil, err
		} else if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}

	return output, err
}

func (c *connectorCommon) GetLogs(ctx context.Context, txHash primitives.Uint256, contractAddress []byte) ([]*types.Log, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	receipt, err := client.TransactionReceipt(ctx, common.BytesToHash(txHash))
	if err != nil {
		return nil, err
	}

	return receipt.Logs, nil
}