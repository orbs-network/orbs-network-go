// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

type harness struct {
	rpcAdapter adapter.DeployingEthereumConnection
	connector  services.CrosschainConnector
	logger     log.Logger
	address    string
	config     *ethereumConnectorConfigForTests
}

func (h *harness) getAddress() string {
	return h.address
}

func (h *harness) deployRpcStorageContract(text string) (string, error) {
	auth, err := h.config.GetAuthFromConfig()
	if err != nil {
		return "", err
	}
	address, err := h.rpcAdapter.DeploySimpleStorageContract(auth, text)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(address[:]), nil
}

func (h *harness) moveBlocksInGanache(t *testing.T, count int, blockGapInSeconds int) {
	c, err := rpc.Dial(h.config.endpoint)
	require.NoError(t, err, "failed creating Ethereum rpc client")
	//start := time.Now()
	for i := 0; i < count; i++ {
		require.NoError(t, c.Call(struct{}{}, "evm_increaseTime", blockGapInSeconds), "failed increasing time")
		require.NoError(t, c.Call(struct{}{}, "evm_mine"), "failed increasing time")
	}

}

func newRpcEthereumConnectorHarness(logger log.Logger, cfg *ethereumConnectorConfigForTests) *harness {
	registry := metric.NewRegistry()
	a := adapter.NewEthereumRpcConnection(cfg, logger, registry)

	return &harness{
		config:     cfg,
		rpcAdapter: a,
		logger:     logger,
		connector:  ethereum.NewEthereumCrosschainConnector(a, cfg, logger, registry),
	}
}

func (h *harness) packInputArgumentsForSampleStorage(method string, args []interface{}) ([]byte, error) {
	if parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI)); err != nil {
		return nil, errors.WithStack(err)
	} else {
		return ethereum.ABIPackFunctionInputArguments(parsedABI, method, args)
	}
}
