// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

var LogTag = log.Service("consensus-context")

type metrics struct {
	createTxBlockTime                         *metric.Histogram
	createResultsBlockTime                    *metric.Histogram
	processTransactionsSeInCreateResultsBlock *metric.Histogram
	transactionsRate                          *metric.Rate
}

func newMetrics(factory metric.Factory) *metrics {
	return &metrics{
		createTxBlockTime:                         factory.NewLatency("ConsensusContext.CreateTransactionsBlockTime.Millis", 10*time.Second),
		createResultsBlockTime:                    factory.NewLatency("ConsensusContext.CreateResultsBlockTime.Millis", 10*time.Second),
		processTransactionsSeInCreateResultsBlock: factory.NewLatency("ConsensusContext.ProcessTransactionsSetInCreateResultsBlock.Millis", 10*time.Second),
		transactionsRate:                          factory.NewRate("ConsensusContext.TransactionsEnteringBlock.PerSecond"),
	}
}

type service struct {
	transactionPool services.TransactionPool
	virtualMachine  services.VirtualMachine
	stateStorage    services.StateStorage
	config          config.ConsensusContextConfig
	logger          log.BasicLogger

	metrics *metrics
}

func NewConsensusContext(
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	stateStorage services.StateStorage,
	config config.ConsensusContextConfig,
	logger log.BasicLogger,
	metricFactory metric.Factory,
) services.ConsensusContext {

	return &service{
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		stateStorage:    stateStorage,
		config:          config,
		logger:          logger.WithTags(LogTag),
		metrics:         newMetrics(metricFactory),
	}
}

func (s *service) RequestNewTransactionsBlock(ctx context.Context, input *services.RequestNewTransactionsBlockInput) (*services.RequestNewTransactionsBlockOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("starting to create transactions block", log.BlockHeight(input.CurrentBlockHeight))
	txBlock, err := s.createTransactionsBlock(ctx, input)
	if err != nil {
		logger.Info("failed to create transactions block", log.Error(err))
		return nil, err
	}

	s.metrics.transactionsRate.Measure(int64(len(txBlock.SignedTransactions)))
	logger.Info("created Transactions block", log.Int("num-transactions", len(txBlock.SignedTransactions)), log.BlockHeight(input.CurrentBlockHeight))
	s.printTxHash(logger, txBlock)
	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil
}

func (s *service) printTxHash(logger log.BasicLogger, txBlock *protocol.TransactionsBlockContainer) {
	for _, tx := range txBlock.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		logger.Info("transaction entered transactions block", log.String("flow", "checkpoint"), log.Transaction(txHash), log.BlockHeight(txBlock.Header.BlockHeight()))
	}
}

func (s *service) RequestNewResultsBlock(ctx context.Context, input *services.RequestNewResultsBlockInput) (*services.RequestNewResultsBlockOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	rxBlock, err := s.createResultsBlock(ctx, input)
	if err != nil {
		return nil, err
	}

	logger.Info("created Results block", log.Int("num-receipts", len(rxBlock.TransactionReceipts)), log.BlockHeight(input.CurrentBlockHeight))

	return &services.RequestNewResultsBlockOutput{
		ResultsBlock: rxBlock,
	}, nil
}
