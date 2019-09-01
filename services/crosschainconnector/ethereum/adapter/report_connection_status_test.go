// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type ethereumConnectorConfigForTests struct {
	endpoint                string
	privateKeyHex           string
	finalityTimeComponent   time.Duration
	finalityBlocksComponent uint32
}

func (c *ethereumConnectorConfigForTests) EthereumEndpoint() string {
	return c.endpoint
}

func (c *ethereumConnectorConfigForTests) EthereumFinalityTimeComponent() time.Duration {
	return c.finalityTimeComponent
}

func (c *ethereumConnectorConfigForTests) EthereumFinalityBlocksComponent() uint32 {
	return c.finalityBlocksComponent
}

func (c *ethereumConnectorConfigForTests) GetAuthFromConfig() (*bind.TransactOpts, error) {
	key, err := crypto.HexToECDSA(c.privateKeyHex)
	if err != nil {
		return nil, err
	}

	return bind.NewKeyedTransactor(key), nil
}

func TestReportingFailure(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {

		emptyConfig := &ethereumConnectorConfigForTests{}
		registry := metric.NewRegistry()
		x := NewEthereumRpcConnection(emptyConfig, harness.Logger, registry)
		err := x.updateConnectionStatus(context.Background(), createConnectionStatusMetrics(registry))
		require.Error(t, err, "require some error from the update flow, config is a lie")
	})
}
