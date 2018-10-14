package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

var LogTag = log.Service("consensus-context")

type metrics struct {
	createTxBlock      *metric.Histogram
	createResultsBlock *metric.Histogram
}

func newMetrics(factory metric.Factory) *metrics {
	return &metrics{
		createTxBlock:      factory.NewLatency("ConsensusContext.createTransactionsBlockTime", 10*time.Second),
		createResultsBlock: factory.NewLatency("ConsensusContext.createResultsBlockTime", 10*time.Second),
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

func (s *service) RequestNewTransactionsBlock(input *services.RequestNewTransactionsBlockInput) (*services.RequestNewTransactionsBlockOutput, error) {
	txBlock, err := s.createTransactionsBlock(input.BlockHeight, input.PrevBlockHash)
	if err != nil {
		return nil, err
	}

	s.logger.Info("created Transactions block", log.Int("num-transactions", len(txBlock.SignedTransactions)), log.Stringable("transactions-block", txBlock))

	for _, tx := range txBlock.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		s.logger.Info("transaction entered transactions block", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash), log.BlockHeight(txBlock.Header.BlockHeight()))
	}

	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil
}

func (s *service) RequestNewResultsBlock(input *services.RequestNewResultsBlockInput) (*services.RequestNewResultsBlockOutput, error) {
	rxBlock, err := s.createResultsBlock(input.BlockHeight, input.PrevBlockHash, input.TransactionsBlock)
	if err != nil {
		return nil, err
	}

	s.logger.Info("created Results block", log.Stringable("results-block", rxBlock))

	return &services.RequestNewResultsBlockOutput{
		ResultsBlock: rxBlock,
	}, nil
}

func (s *service) ValidateTransactionsBlock(input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateResultsBlock(input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) RequestOrderingCommittee(input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	panic("Not implemented")
}

func (s *service) RequestValidationCommittee(input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	panic("Not implemented")
}
