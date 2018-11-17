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
	txBlock, err := s.createTransactionsBlock(ctx, input.BlockHeight, input.PrevBlockHash)
	if err != nil {
		return nil, err
	}

	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil
}

func (s *service) printTxHash(txBlock *protocol.TransactionsBlockContainer) {
	for _, tx := range txBlock.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		s.logger.Info("transaction entered transactions block", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash), log.BlockHeight(txBlock.Header.BlockHeight()))
	}
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
		return nil, errors.New(fmt.Sprintf("Incorrect protocol version: needed %v but received %v", requiredProtocolVersion, blockProtocolVersion))
	}
	if blockVirtualChainId != requiredVirtualChainId {
		return nil, errors.New(fmt.Sprintf("Incorrect virtual chain ID: needed %v but received %v", requiredVirtualChainId, blockVirtualChainId))
	}
	calculatedTxRoot, err := CalculateTransactionsRootHash(txs)
	if err != nil {
		return nil, err
	}

	calculatedPrevBlockHashPtr := CalculatePrevBlockHashPtr(input.TransactionsBlock)

	if bytes.Compare(blockTxRoot, calculatedTxRoot) != 0 {
		return nil, errors.New("incorrect transactions root hash")
	}
	if bytes.Compare(prevBlockHashPtr, calculatedPrevBlockHashPtr) != 0 {
		return nil, errors.New("incorrect previous block hash")
	}

	return &services.ValidateTransactionsBlockOutput{}, nil

}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	panic("Not implemented")
}
