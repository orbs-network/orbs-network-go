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
	checkedHeader := input.TransactionsBlock.Header
	expectedProtocolVersion := s.config.ProtocolVersion()
	expectedVirtualChainId := s.config.VirtualChainId()

	txs := input.TransactionsBlock.SignedTransactions
	txMerkleRootHash := checkedHeader.TransactionsRootHash()

	prevBlockHashPtr := checkedHeader.PrevBlockHashPtr()

	blockProtocolVersion := checkedHeader.ProtocolVersion()
	blockVirtualChainId := checkedHeader.VirtualChainId()

	if blockProtocolVersion != expectedProtocolVersion {
		return nil, fmt.Errorf("incorrect protocol version: expected %v but block has %v", expectedProtocolVersion, blockProtocolVersion)
	}
	if blockVirtualChainId != expectedVirtualChainId {
		return nil, fmt.Errorf("incorrect virtual chain ID: expected %v but block has %v", expectedVirtualChainId, blockVirtualChainId)
	}
	if input.BlockHeight != checkedHeader.BlockHeight() {
		return nil, fmt.Errorf("mismatching blockHeight: input %v checkedHeader %v", input.BlockHeight, checkedHeader.BlockHeight())
	}
	calculatedTxRoot, err := CalculateTransactionsRootHash(txs)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(txMerkleRootHash, calculatedTxRoot) {
		return nil, errors.New("incorrect transactions root hash")
	}
	calculatedPrevBlockHashPtr := CalculatePrevBlockHashPtr(input.TransactionsBlock)
	if !bytes.Equal(prevBlockHashPtr, calculatedPrevBlockHashPtr) {
		return nil, errors.New("incorrect previous block hash")
	}

	// TODO "Check timestamp is within configurable allowed jitter of system timestamp, and later than previous block"

	// TODO "Check transaction merkle root hash" https://github.com/orbs-network/orbs-spec/issues/118

	// TODO "Check metadata hash"

	validationInput := &services.ValidateTransactionsForOrderingInput{
		BlockHeight:        input.BlockHeight,
		BlockTimestamp:     input.TransactionsBlock.Header.Timestamp(),
		SignedTransactions: input.TransactionsBlock.SignedTransactions,
	}

	_, err = s.transactionPool.ValidateTransactionsForOrdering(ctx, validationInput)
	if err != nil {
		return nil, err
	}

	return &services.ValidateTransactionsBlockOutput{}, nil

}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	expectedProtocolVersion := s.config.ProtocolVersion()
	expectedVirtualChainId := s.config.VirtualChainId()

	checkedHeader := input.ResultsBlock.Header
	blockProtocolVersion := checkedHeader.ProtocolVersion()
	blockVirtualChainId := checkedHeader.VirtualChainId()
	if blockProtocolVersion != expectedProtocolVersion {
		return nil, fmt.Errorf("incorrect protocol version: expected %v but block has %v", expectedProtocolVersion, blockProtocolVersion)
	}
	if blockVirtualChainId != expectedVirtualChainId {
		return nil, fmt.Errorf("incorrect virtual chain ID: expected %v but block has %v", expectedVirtualChainId, blockVirtualChainId)
	}
	if input.BlockHeight != checkedHeader.BlockHeight() {
		return nil, fmt.Errorf("mismatching blockHeight: input %v checkedHeader %v", input.BlockHeight, checkedHeader.BlockHeight())
	}

	prevBlockHashPtr := input.ResultsBlock.Header.PrevBlockHashPtr()
	if !bytes.Equal(input.PrevBlockHash, prevBlockHashPtr) {
		return nil, errors.New("incorrect previous results block hash")
	}
	if checkedHeader.Timestamp() != input.TransactionsBlock.Header.Timestamp() {
		return nil, fmt.Errorf("mismatching timestamps: txBlock=%v rxBlock=%v", checkedHeader.Timestamp(), input.TransactionsBlock.Header.Timestamp())
	}
	// Check the receipts merkle root matches the receipts.
	receipts := input.ResultsBlock.TransactionReceipts
	calculatedReceiptsRoot, err := CalculateReceiptsRootHash(receipts)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(checkedHeader.ReceiptsRootHash(), calculatedReceiptsRoot) {
		return nil, errors.New("incorrect receipts root hash")
	}

	// Check the hash of the state diff in the block.
	// TODO Statediff not impl - pending https://github.com/orbs-network/orbs-spec/issues/111

	// Check hash pointer to the Transactions block of the same height.
	// TODO Then what is input.BlockHeight() - yet another block height of something??
	if checkedHeader.BlockHeight() != input.TransactionsBlock.Header.BlockHeight() {
		return nil, fmt.Errorf("mismatching block height: txBlock=%v rxBlock=%v", checkedHeader.BlockHeight(), input.TransactionsBlock.Header.BlockHeight())
	}

	// Check merkle root of the state prior to the block execution, retrieved by calling `StateStorage.GetStateHash`.

	calculatedPreExecutionStateRootHash, err := s.stateStorage.GetStateHash(ctx, &services.GetStateHashInput{
		BlockHeight: checkedHeader.BlockHeight(),
	})
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(checkedHeader.PreExecutionStateRootHash(), calculatedPreExecutionStateRootHash.StateRootHash) {
		return nil, fmt.Errorf("mismatching PreExecutionStateRootHash: expected %v but results block hash %v",
			calculatedPreExecutionStateRootHash, checkedHeader.PreExecutionStateRootHash())
	}

	// Check transaction id bloom filter (see block format for structure).
	// TODO Pending spec https://github.com/orbs-network/orbs-spec/issues/118

	// Check transaction timestamp bloom filter (see block format for structure).
	// TODO Pending spec https://github.com/orbs-network/orbs-spec/issues/118

	// Validate transaction execution

	// Execute the ordered transactions set by calling VirtualMachine.ProcessTransactionSet
	// (creating receipts and state diff). Using the provided header timestamp as a reference timestamp.
	_, err = s.virtualMachine.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		BlockHeight:        checkedHeader.BlockHeight(),
		SignedTransactions: input.TransactionsBlock.SignedTransactions,
	})
	if err != nil {
		return nil, err
	}

	// Compare the receipts merkle root hash to the one in the block

	// Compare the state diff hash to the one in the block (supports only deterministic execution).

	// TODO How to calculate receipts merkle hash root and state diff hash
	// See https://github.com/orbs-network/orbs-spec/issues/111
	//blockMerkleRootHash := checkedHeader.ReceiptsRootHash()

	return &services.ValidateResultsBlockOutput{}, nil

}
