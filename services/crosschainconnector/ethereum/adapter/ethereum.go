package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type ethereumAdapterConfig interface {
	EthereumEndpoint() string
}

type EthereumConnection interface {
	CallContract(ctx context.Context, address []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error)
}

type connectorCommon struct {
	getContractCaller func() (bind.ContractBackend, error)
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


