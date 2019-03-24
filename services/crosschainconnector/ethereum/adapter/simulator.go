// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"math/big"
	"sync"
)

type EthereumSimulator struct {
	connectorCommon

	auth *bind.TransactOpts
	mu   struct {
		sync.Mutex
		simClient *backends.SimulatedBackend
	}
}

func NewEthereumSimulatorConnection(logger log.BasicLogger) *EthereumSimulator {
	// Generate a new random account and a funded simulator
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err.Error())
	}

	e := &EthereumSimulator{
		auth: bind.NewKeyedTransactor(key),
	}

	e.auth.GasLimit = 999999999
	e.auth.GasPrice = big.NewInt(1)

	e.logger = logger.WithTags(log.String("adapter", "ethereum-sim"))

	e.getContractCaller = func() (EthereumCaller, error) {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.mu.simClient == nil {
			e.createClientAndInitAccount()
		}

		return e.mu.simClient, nil
	}

	return e
}

func (es *EthereumSimulator) createClientAndInitAccount() {

	genesisAllocation := map[common.Address]core.GenesisAccount{
		es.auth.From: {Balance: big.NewInt(10000000000)},
	}

	es.mu.simClient = backends.NewSimulatedBackend(genesisAllocation, 900000000000)

}

func (es *EthereumSimulator) GetAuth() *bind.TransactOpts {
	// this is used for test code, not protecting this
	return es.auth
}

func (es *EthereumSimulator) Commit() {
	es.mu.simClient.Commit()
}

func (es *EthereumSimulator) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, nil
}

func (c *EthereumSimulator) ReportConnectionStatus(ctx context.Context, registry metric.Registry, logger log.BasicLogger) {
}
