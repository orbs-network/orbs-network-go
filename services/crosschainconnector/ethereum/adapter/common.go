// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"math/big"
)

type ethereumAdapterConfig interface {
	EthereumEndpoint() string
}

type EthereumConnection interface {
	CallContract(ctx context.Context, contractAddress []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error)
	GetTransactionLogs(ctx context.Context, txHash primitives.Uint256, eventSignature []byte) ([]*TransactionLog, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	ReportConnectionStatus(ctx context.Context, registry metric.Registry, logger log.BasicLogger)
}

type connectorCommon struct {
	logger            log.BasicLogger
	getContractCaller func() (EthereumCaller, error)
}

type EthereumCaller interface {
	bind.ContractBackend
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

func (c *connectorCommon) CallContract(ctx context.Context, contractAddress []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	address := common.BytesToAddress(contractAddress)

	// we do not support pending calls, opts is always empty
	opts := new(bind.CallOpts)

	msg := ethereum.CallMsg{From: opts.From, To: &address, Data: packedInput}
	output, err := client.CallContract(ctx, msg, blockNumber)
	if err == nil && len(output) == 0 {
		// make sure we have a contract to operate on, and bail out otherwise.
		if code, err := client.CodeAt(ctx, address, blockNumber); err != nil {
			return nil, err
		} else if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}

	return output, err
}

func (c *connectorCommon) Receipt(txHash common.Hash) (*types.Receipt, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	return client.TransactionReceipt(context.TODO(), txHash)
}

func (c *connectorCommon) SyncProgress() (*ethereum.SyncProgress, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	if ethClient, ok := client.(*ethclient.Client); ok {
		return ethClient.SyncProgress(context.TODO())
	}

	return nil, nil
}
