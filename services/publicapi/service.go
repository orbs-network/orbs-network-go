// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"time"
)

var LogTag = log.Service("public-api")

type service struct {
	config          config.PublicApiConfig
	transactionPool services.TransactionPool
	virtualMachine  services.VirtualMachine
	blockStorage    services.BlockStorage
	logger          log.Logger

	waiter *waiter

	metrics *metrics
}

type metrics struct {
	sendTransactionTime                *metric.Histogram
	getTransactionStatusTime           *metric.Histogram
	runQueryTime                       *metric.Histogram
	totalTransactionsFromClients       *metric.Gauge
	totalTransactionsErrNilRequest     *metric.Gauge
	totalTransactionsErrInvalidRequest *metric.Gauge
	totalTransactionsErrAddingToTxPool *metric.Gauge
	totalTransactionsErrDuplicate      *metric.Gauge
	queriesPerSecond                   *metric.Rate
	transactionsPerSecond              *metric.Rate
}

func newMetrics(factory metric.Factory, sendTransactionTimeout time.Duration, getTransactionStatusTimeout time.Duration, runQueryTimeout time.Duration) *metrics {
	return &metrics{
		sendTransactionTime:                factory.NewLatency("PublicApi.SendTransactionProcessingTime.Millis", sendTransactionTimeout),
		getTransactionStatusTime:           factory.NewLatency("PublicApi.GetTransactionStatusProcessingTime.Millis", getTransactionStatusTimeout),
		runQueryTime:                       factory.NewLatency("PublicApi.RunQueryProcessingTime.Millis", runQueryTimeout),
		totalTransactionsFromClients:       factory.NewGauge("PublicApi.TotalTransactionsFromClients.Count"),
		totalTransactionsErrNilRequest:     factory.NewGauge("PublicApi.TotalTransactionsErrNilRequest.Count"),
		totalTransactionsErrInvalidRequest: factory.NewGauge("PublicApi.TotalTransactionsErrInvalidRequest.Count"),
		totalTransactionsErrAddingToTxPool: factory.NewGauge("PublicApi.TotalTransactionsErrAddingToTxPool.Count"),
		totalTransactionsErrDuplicate:      factory.NewGauge("PublicApi.TotalTransactionsErrDuplicate.Count"),
		queriesPerSecond:                   factory.NewRate("PublicApi.Queries"),
		transactionsPerSecond:              factory.NewRate("PublicApi.Transactions"),
	}
}

func NewPublicApi(
	config config.PublicApiConfig,
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	blockStorage services.BlockStorage,
	logger log.Logger,
	metricFactory metric.Factory,
) services.PublicApi {
	s := &service{
		config:          config,
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		blockStorage:    blockStorage,
		logger:          logger.WithTags(LogTag),

		waiter:  newWaiter(),
		metrics: newMetrics(metricFactory, config.PublicApiSendTransactionTimeout(), 2*time.Second, 1*time.Second),
	}

	transactionPool.RegisterTransactionResultsHandler(s)

	return s
}

func (s *service) HandleTransactionResults(ctx context.Context, input *handlers.HandleTransactionResultsInput) (*handlers.HandleTransactionResultsOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.String("flow", "checkpoint"))

	for _, txReceipt := range input.TransactionReceipts {
		logger.Info("transaction reported as committed", logfields.Transaction(txReceipt.Txhash()))
		s.waiter.complete(txReceipt.Txhash().KeyForMap(),
			&txOutput{
				transactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
				transactionReceipt: txReceipt,
				blockHeight:        input.BlockHeight,
				blockTimestamp:     input.Timestamp,
			})
	}
	return &handlers.HandleTransactionResultsOutput{}, nil
}

func (s *service) HandleTransactionError(ctx context.Context, input *handlers.HandleTransactionErrorInput) (*handlers.HandleTransactionErrorOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.String("flow", "checkpoint"))

	logger.Info("transaction reported as erred", logfields.Transaction(input.Txhash), log.Stringable("tx-status", input.TransactionStatus))
	s.waiter.complete(input.Txhash.KeyForMap(),
		&txOutput{
			transactionStatus:  input.TransactionStatus,
			transactionReceipt: nil,
			blockHeight:        input.BlockHeight,
			blockTimestamp:     input.BlockTimestamp,
		})
	return &handlers.HandleTransactionErrorOutput{}, nil
}
