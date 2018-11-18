package consensuscontext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	createTxBlockTime      *metric.Histogram
	createResultsBlockTime *metric.Histogram
	transactionsRate       *metric.Rate
}

func newMetrics(factory metric.Factory) *metrics {
	return &metrics{
		createTxBlockTime:      factory.NewLatency("ConsensusContext.CreateTransactionsBlockTime", 10*time.Second),
		createResultsBlockTime: factory.NewLatency("ConsensusContext.CreateResultsBlockTime", 10*time.Second),
		transactionsRate:       factory.NewRate("ConsensusContext.TransactionsPerSecond"),
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
	logger.Info("starting to create transactions block")
	txBlock, err := s.createTransactionsBlock(ctx, input.BlockHeight, input.PrevBlockHash)
	if err != nil {
		logger.Info("failed to create transactions block", log.Error(err))
		return nil, err
	}

	s.metrics.transactionsRate.Measure(int64(len(txBlock.SignedTransactions)))
	logger.Info("created transactions block", log.Int("num-transactions", len(txBlock.SignedTransactions)), log.Stringable("transactions-block", txBlock))
	s.printTxHash(logger, txBlock)
	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil
}

func (s *service) printTxHash(logger log.BasicLogger, txBlock *protocol.TransactionsBlockContainer) {
	for _, tx := range txBlock.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		logger.Info("transaction entered transactions block", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash), log.BlockHeight(txBlock.Header.BlockHeight()))
	}
}

func (s *service) RequestNewResultsBlock(ctx context.Context, input *services.RequestNewResultsBlockInput) (*services.RequestNewResultsBlockOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	rxBlock, err := s.createResultsBlock(ctx, input.BlockHeight, input.PrevBlockHash, input.TransactionsBlock)
	if err != nil {
		return nil, err
	}

	logger.Info("created Results block", log.Stringable("results-block", rxBlock))

	return &services.RequestNewResultsBlockOutput{
		ResultsBlock: rxBlock,
	}, nil
}

func (s *service) ValidateTransactionsBlock(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {

	// TODO maybe put these validations in a table
	requiredProtocolVersion := s.config.ProtocolVersion()
	requiredVirtualChainId := s.config.VirtualChainId()

	header := input.TransactionsBlock.Header
	txs := input.TransactionsBlock.SignedTransactions
	blockTxRoot := input.TransactionsBlock.Header.TransactionsRootHash()
	prevBlockHashPtr := input.TransactionsBlock.Header.PrevBlockHashPtr()

	blockProtocolVersion := header.ProtocolVersion()
	blockVirtualChainId := header.VirtualChainId()

	if blockProtocolVersion != requiredProtocolVersion {
		return nil, fmt.Errorf("incorrect protocol version: needed %v but received %v", requiredProtocolVersion, blockProtocolVersion)
	}
	if blockVirtualChainId != requiredVirtualChainId {
		return nil, fmt.Errorf("incorrect virtual chain ID: needed %v but received %v", requiredVirtualChainId, blockVirtualChainId)
	}
	calculatedTxRoot, err := CalculateTransactionsRootHash(txs)
	if err != nil {
		return nil, err
	}

	calculatedPrevBlockHashPtr := CalculatePrevBlockHashPtr(input.TransactionsBlock)

	if !bytes.Equal(blockTxRoot, calculatedTxRoot) {
		return nil, errors.New("incorrect transactions root hash")
	}
	if !bytes.Equal(prevBlockHashPtr, calculatedPrevBlockHashPtr) {
		return nil, errors.New("incorrect previous block hash")
	}

	return &services.ValidateTransactionsBlockOutput{}, nil

}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	panic("Not implemented")
}
