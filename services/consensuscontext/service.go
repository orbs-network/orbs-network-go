package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
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
	txBlock, err := s.createTransactionsBlock(ctx, input.BlockHeight, input.PrevBlockHash)
	if err != nil {
		return nil, err
	}

	s.logger.Info("created Transactions block", log.Int("num-transactions", len(txBlock.SignedTransactions)), log.Stringable("transactions-block", txBlock))

	s.metrics.transactionsRate.Measure(int64(len(txBlock.SignedTransactions)))

	for _, tx := range txBlock.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		s.logger.Info("transaction entered transactions block", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash), log.BlockHeight(txBlock.Header.BlockHeight()))
	}

	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil
}

func (s *service) RequestNewResultsBlock(ctx context.Context, input *services.RequestNewResultsBlockInput) (*services.RequestNewResultsBlockOutput, error) {
	rxBlock, err := s.createResultsBlock(ctx, input.BlockHeight, input.PrevBlockHash, input.TransactionsBlock)
	if err != nil {
		return nil, err
	}

	s.logger.Info("created Results block", log.Stringable("results-block", rxBlock))

	return &services.RequestNewResultsBlockOutput{
		ResultsBlock: rxBlock,
	}, nil
}

func (s *service) ValidateTransactionsBlock(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) RequestOrderingCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	panic("Not implemented")
}

func (s *service) RequestValidationCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	panic("Not implemented")
}

func CalculateCommitteeSize(requestedCommitteeSize int, federationSize int) int {

	if requestedCommitteeSize > federationSize {
		return federationSize
	}
	return requestedCommitteeSize
}

// Smart algo!
func ChooseRandomCommitteeIndices(input *services.RequestCommitteeInput) []int {
	indices := make([]int, input.MaxCommitteeSize)
	for i := 0; i < int(input.MaxCommitteeSize); i++ {
		indices[i] = i
	}
	return indices
}
