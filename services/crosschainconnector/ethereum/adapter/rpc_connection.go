// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"math/big"
)

type EthereumRpcConnection struct {
	govnr.TreeSupervisor
	connectorCommon

	config   ethereumAdapterConfig
	registry metric.Registry
}

func NewEthereumRpcConnection(config ethereumAdapterConfig, logger log.Logger, registry metric.Registry) *EthereumRpcConnection {
	rpc := &EthereumRpcConnection{
		connectorCommon: connectorCommon{
			logger: logger.WithTags(log.String("adapter", "ethereum")),
		},
		config:   config,
		registry: registry,
	}
	rpc.getContractCaller = func() (caller EthereumCaller, e error) {
		return rpc.dial()
	}
	return rpc
}

func (rpc *EthereumRpcConnection) dial() (*ethclient.Client, error) {
	return ethclient.Dial(rpc.config.EthereumEndpoint())
}

func (rpc *EthereumRpcConnection) HeaderByNumber(ctx context.Context, number *big.Int) (*BlockNumberAndTime, error) {
	client, err := rpc.dial()
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

	return &BlockNumberAndTime{
		TimeInSeconds: header.Time,
		BlockNumber:   header.Number.Int64(),
	}, nil
}
