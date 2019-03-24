// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"time"
)

type metrics struct {
	syncStatus              *metric.Text
	lastBlock               *metric.Gauge
	receiptsRetrievalStatus *metric.Text
}

const STATUS_FAILED = "failed"
const STATUS_SUCCESS = "success"
const STATUS_IN_PROGRESS = "in-progress"

const ARBITRARY_TXHASH = "0xb41e0591756bd1331de35eac3e3da460c9b3503d10e7bf08b84f057f489cd189"

func (c *EthereumRpcConnection) ReportConnectionStatus(ctx context.Context, registry metric.Registry, logger log.BasicLogger) {
	metrics := &metrics{
		syncStatus:              registry.NewText("Ethereum.Node.Sync.Status", STATUS_FAILED),
		lastBlock:               registry.NewGauge("Ethereum.Node.LastBlock"),
		receiptsRetrievalStatus: registry.NewText("Ethereum.Node.TransactionReceipts.Status", STATUS_FAILED),
	}

	synchronization.NewPeriodicalTrigger(ctx, 30*time.Second, logger, func() {
		if receipt, err := c.Receipt(common.HexToHash(ARBITRARY_TXHASH)); err != nil {
			logger.Info("ethereum rpc connection status check failed", log.Error(err))
			metrics.receiptsRetrievalStatus.Update(STATUS_FAILED)
		} else if len(receipt.Logs) > 0 {
			metrics.receiptsRetrievalStatus.Update(STATUS_SUCCESS)
		} else {
			metrics.receiptsRetrievalStatus.Update(STATUS_FAILED)
		}

		if syncStatus, err := c.SyncProgress(); err != nil {
			logger.Info("ethereum rpc connection status check failed", log.Error(err))
			metrics.syncStatus.Update(STATUS_FAILED)
		} else if syncStatus == nil {
			metrics.syncStatus.Update(STATUS_SUCCESS)
		} else {
			metrics.syncStatus.Update(STATUS_IN_PROGRESS)
		}

		if header, err := c.HeaderByNumber(ctx, nil); err != nil {
			logger.Info("ethereum rpc connection status check failed", log.Error(err))
			metrics.lastBlock.Update(0)
		} else {
			metrics.lastBlock.Update(header.Number.Int64())
		}
	}, nil)
}
