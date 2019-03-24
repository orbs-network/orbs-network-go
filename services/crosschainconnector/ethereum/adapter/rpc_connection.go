// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"math/big"
	"sync"
)

type EthereumRpcConnection struct {
	connectorCommon

	config ethereumAdapterConfig

	mu struct {
		sync.Mutex
		client *ethclient.Client
	}
}

func NewEthereumRpcConnection(config ethereumAdapterConfig, logger log.BasicLogger) *EthereumRpcConnection {
	rpc := &EthereumRpcConnection{
		config: config,
	}
	rpc.logger = logger.WithTags(log.String("adapter", "ethereum"))
	rpc.getContractCaller = func() (caller EthereumCaller, e error) {
		return rpc.dialIfNeededAndReturnClient()
	}
	return rpc
}

func (rpc *EthereumRpcConnection) dial() error {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	if rpc.mu.client != nil {
		return nil
	}
	if client, err := ethclient.Dial(rpc.config.EthereumEndpoint()); err != nil {
		return err
	} else {
		rpc.mu.client = client
	}
	return nil
}

func (rpc *EthereumRpcConnection) dialIfNeededAndReturnClient() (*ethclient.Client, error) {
	if rpc.mu.client == nil {
		if err := rpc.dial(); err != nil {
			return nil, err
		}
	}
	return rpc.mu.client, nil
}

func (rpc *EthereumRpcConnection) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	client, err := rpc.dialIfNeededAndReturnClient()
	if err != nil {
		return nil, err
	}

	header, err := client.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	// not supposed to happen since client.HeaderByNumber does not return nil, nil
	if header == nil {
		return nil, errors.New("ethereum returned nil header without error")
	}

	return header, nil
}
