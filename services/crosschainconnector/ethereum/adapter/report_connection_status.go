// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/scribe/log"
	"time"
)

type metrics struct {
	syncStatus              *metric.Text
	lastBlock               *metric.Gauge
	receiptsRetrievalStatus *metric.Text
	endpoint                *metric.Text
}

const STATUS_FAILED = "failed"
const STATUS_SUCCESS = "success"
const STATUS_IN_PROGRESS = "in-progress"

const ARBITRARY_TXHASH = "0xb41e0591756bd1331de35eac3e3da460c9b3503d10e7bf08b84f057f489cd189"

func createConnectionStatusMetrics(registry metric.Registry) *metrics {
	statusMetrics := &metrics{
		syncStatus:              registry.NewText("Ethereum.Node.Sync.Status", STATUS_FAILED),
		lastBlock:               registry.NewGauge("Ethereum.Node.LastBlock"),
		receiptsRetrievalStatus: registry.NewText("Ethereum.Node.TransactionReceipts.Status", STATUS_FAILED),
		endpoint:                registry.NewText("Ethereum.Node.Endpoint.Address", ""),
	}

	return statusMetrics
}

func (c *EthereumRpcConnection) ReportConnectionStatus(ctx context.Context) {
	statusMetrics := createConnectionStatusMetrics(c.registry)
	statusMetrics.endpoint.Update(c.config.EthereumEndpoint())

	c.Supervise(synchronization.NewPeriodicalTrigger(ctx, "Ethereum connector status reporter", synchronization.NewTimeTicker(30*time.Second), c.logger, func() {
		if err := c.updateConnectionStatus(ctx, statusMetrics); err != nil {
			c.logger.Info("ethereum rpc connection status check failed", log.Error(err))
		}
	}, nil))
}

func (c *EthereumRpcConnection) updateConnectionStatus(ctx context.Context, m *metrics) error {
	// we always run all checks, and return an error in any of them - its the metrics that matter
	var ethError error
	if receipt, err := c.Receipt(common.HexToHash(ARBITRARY_TXHASH)); err != nil {
		ethError = err
		m.receiptsRetrievalStatus.Update(STATUS_FAILED)
	} else if len(receipt.Logs) > 0 {
		m.receiptsRetrievalStatus.Update(STATUS_SUCCESS)
	} else {
		m.receiptsRetrievalStatus.Update(STATUS_FAILED)
	}

	if syncStatus, err := c.SyncProgress(); err != nil {
		ethError = err
		m.syncStatus.Update(STATUS_FAILED)
	} else if syncStatus == nil {
		m.syncStatus.Update(STATUS_SUCCESS)
	} else {
		m.syncStatus.Update(STATUS_IN_PROGRESS)
	}

	if header, err := c.HeaderByNumber(ctx, nil); err != nil {
		ethError = err
		m.lastBlock.Update(0)
	} else {
		m.lastBlock.Update(header.BlockNumber)
	}

	return ethError
}
