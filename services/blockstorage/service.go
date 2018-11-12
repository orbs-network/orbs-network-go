package blockstorage

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/bloom"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	extSync "github.com/orbs-network/orbs-network-go/services/blockstorage/externalsync"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

const (
	// TODO extract it to the spec
	ProtocolVersion = primitives.ProtocolVersion(1)
)

var LogTag = log.Service("block-storage")

type service struct {
	persistence  adapter.BlockPersistence
	stateStorage services.StateStorage
	gossip       gossiptopics.BlockSync
	txPool       services.TransactionPool

	config config.BlockStorageConfig

	logger                  log.BasicLogger
	consensusBlocksHandlers []handlers.ConsensusBlocksHandler

	// lastCommittedBlock state variable is inside adapter.BlockPersistence (GetLastBlock)

	extSync *extSync.BlockSync

	metrics *metrics
}

type metrics struct {
	blockHeight *metric.Gauge
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		blockHeight: m.NewGauge("BlockStorage.BlockHeight"),
	}
}

func NewBlockStorage(ctx context.Context, config config.BlockStorageConfig, persistence adapter.BlockPersistence, stateStorage services.StateStorage, gossip gossiptopics.BlockSync,
	txPool services.TransactionPool, logger log.BasicLogger, metricFactory metric.Factory) services.BlockStorage {
	s := &service{
		persistence:  persistence,
		stateStorage: stateStorage,
		gossip:       gossip,
		txPool:       txPool,
		logger:       logger.WithTags(LogTag),
		config:       config,
		metrics:      newMetrics(metricFactory),
	}

	gossip.RegisterBlockSyncHandler(s)
	s.extSync = extSync.NewExtBlockSync(ctx, config, gossip, s, logger, metricFactory)

	return s
}

func (s *service) GetLastCommittedBlockHeight(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
	b, err := s.persistence.GetLastBlock()
	if err != nil {
		return nil, err
	}
	return &services.GetLastCommittedBlockHeightOutput{
		LastCommittedBlockHeight:    getBlockHeight(b),
		LastCommittedBlockTimestamp: getBlockTimestamp(b),
	}, nil
}

func (s *service) CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	txBlockHeader := input.BlockPair.TransactionsBlock.Header
	rsBlockHeader := input.BlockPair.ResultsBlock.Header

	s.logger.Info("Trying to commit a block", log.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.validateProtocolVersion(input.BlockPair); err != nil {
		return nil, err
	}

	// the source of truth for the last committed block is persistence
	lastCommittedBlock, err := s.persistence.GetLastBlock()
	if err != nil {
		return nil, err
	}

	if ok, err := s.validateBlockDoesNotExist(txBlockHeader, rsBlockHeader, lastCommittedBlock); err != nil || !ok {
		return nil, err
	}

	if err := s.validateBlockHeight(input.BlockPair, lastCommittedBlock); err != nil {
		return nil, err
	}

	if err := s.persistence.WriteNextBlock(input.BlockPair); err != nil {
		return nil, err
	}

	s.metrics.blockHeight.Update(int64(input.BlockPair.TransactionsBlock.Header.BlockHeight()))

	s.extSync.HandleBlockCommitted(ctx)

	s.logger.Info("committed a block", log.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.syncBlockToStateStorage(ctx, input.BlockPair); err != nil {
		// TODO: since the internal-node sync flow is self healing, we should not fail the entire commit if state storage is slow to sync
		s.logger.Error("internal-node sync to state storage failed", log.Error(err))
	}

	if err := s.syncBlockToTxPool(ctx, input.BlockPair); err != nil {
		// TODO: since the internal-node sync flow is self healing, should we fail if pool fails ?
		s.logger.Error("internal-node sync to tx pool failed", log.Error(err))
	}

	return nil, nil
}

func getBlockHeight(block *protocol.BlockPairContainer) primitives.BlockHeight {
	if block == nil {
		return 0
	}
	return block.TransactionsBlock.Header.BlockHeight()
}

func getBlockTimestamp(block *protocol.BlockPairContainer) primitives.TimestampNano {
	if block == nil {
		return 0
	}
	return block.TransactionsBlock.Header.Timestamp()
}

func (s *service) loadTransactionsBlockHeader(height primitives.BlockHeight) (*services.GetTransactionsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetTransactionsBlock(height)
	if err != nil {
		return nil, err
	}
	return &services.GetTransactionsBlockHeaderOutput{
		TransactionsBlockProof:    txBlock.BlockProof,
		TransactionsBlockHeader:   txBlock.Header,
		TransactionsBlockMetadata: txBlock.Metadata,
	}, nil
}

func (s *service) GetTransactionsBlockHeader(ctx context.Context, input *services.GetTransactionsBlockHeaderInput) (result *services.GetTransactionsBlockHeaderOutput, err error) {
	err = s.persistence.GetBlockTracker().WaitForBlock(ctx, input.BlockHeight)
	if err == nil {
		return s.loadTransactionsBlockHeader(input.BlockHeight)
	}
	return nil, err
}

func (s *service) loadResultsBlockHeader(height primitives.BlockHeight) (*services.GetResultsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetResultsBlock(height)
	if err != nil {
		return nil, err
	}
	return &services.GetResultsBlockHeaderOutput{
		ResultsBlockProof:  txBlock.BlockProof,
		ResultsBlockHeader: txBlock.Header,
	}, nil
}

func (s *service) GetResultsBlockHeader(ctx context.Context, input *services.GetResultsBlockHeaderInput) (result *services.GetResultsBlockHeaderOutput, err error) {
	err = s.persistence.GetBlockTracker().WaitForBlock(ctx, input.BlockHeight)
	if err == nil {
		return s.loadResultsBlockHeader(input.BlockHeight)
	}
	return nil, err
}

func (s *service) createEmptyTransactionReceiptResult(ctx context.Context) (*services.GetTransactionReceiptOutput, error) {
	out, err := s.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return nil, err
	}
	return &services.GetTransactionReceiptOutput{
		TransactionReceipt: nil,
		BlockHeight:        out.LastCommittedBlockHeight,
		BlockTimestamp:     out.LastCommittedBlockTimestamp,
	}, nil
}

// TODO: are we sure that if we don't find the receipt this API should fail? it should succeed just return nil receipt
func (s *service) GetTransactionReceipt(ctx context.Context, input *services.GetTransactionReceiptInput) (*services.GetTransactionReceiptOutput, error) {
	searchRules := adapter.BlockSearchRules{
		EndGraceNano:          s.config.BlockTransactionReceiptQueryGraceEnd().Nanoseconds(),
		StartGraceNano:        s.config.BlockTransactionReceiptQueryGraceStart().Nanoseconds(),
		TransactionExpireNano: s.config.BlockTransactionReceiptQueryExpirationWindow().Nanoseconds(),
	}
	blocksToSearch := s.persistence.GetBlocksRelevantToTxTimestamp(input.TransactionTimestamp, searchRules)
	if blocksToSearch == nil {
		receipt, err := s.createEmptyTransactionReceiptResult(ctx)
		if err != nil {
			return nil, err
		}
		// TODO: probably don't fail here (issue#448)
		return receipt, errors.Errorf("failed to search for blocks on tx timestamp of %d, hash %s", input.TransactionTimestamp, input.Txhash)
	}

	if len(blocksToSearch) == 0 {
		// duplication of this piece of code is a smell originating from issue#448
		receipt, err := s.createEmptyTransactionReceiptResult(ctx)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	}

	for _, b := range blocksToSearch {
		tbf := bloom.NewFromRaw(b.ResultsBlock.Header.TimestampBloomFilter())
		if tbf.Test(input.TransactionTimestamp) {
			for _, txr := range b.ResultsBlock.TransactionReceipts {
				if txr.Txhash().Equal(input.Txhash) {
					return &services.GetTransactionReceiptOutput{
						TransactionReceipt: txr,
						BlockHeight:        b.ResultsBlock.Header.BlockHeight(),
						BlockTimestamp:     b.ResultsBlock.Header.Timestamp(),
					}, nil
				}
			}
		}
	}

	receipt, err := s.createEmptyTransactionReceiptResult(ctx)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// FIXME implement all block checks
func (s *service) ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	if protocolVersionError := s.validateProtocolVersion(input.BlockPair); protocolVersionError != nil {
		return nil, protocolVersionError
	}

	// the source of truth for the last committed block is persistence
	lastCommittedBlock, err := s.persistence.GetLastBlock()
	if err != nil {
		return nil, err
	}

	if blockHeightError := s.validateBlockHeight(input.BlockPair, lastCommittedBlock); blockHeightError != nil {
		return nil, blockHeightError
	}

	if err := s.validateWithConsensusAlgosWithMode(
		ctx,
		lastCommittedBlock,
		input.BlockPair,
		handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE); err != nil {

		s.logger.Error("internal-node sync to consensus algo failed", log.Error(err))
	}

	return &services.ValidateBlockForCommitOutput{}, nil
}

func (s *service) RegisterConsensusBlocksHandler(handler handlers.ConsensusBlocksHandler) {
	s.consensusBlocksHandlers = append(s.consensusBlocksHandlers, handler)

	// update the consensus algo about the latest block we have (for its initialization)
	s.UpdateConsensusAlgosAboutLatestCommittedBlock(context.TODO()) // TODO: (talkol) not sure if we should create a new context here or pass to RegisterConsensusBlocksHandler in code generation
}

// TODO: this function should return an error
func (s *service) UpdateConsensusAlgosAboutLatestCommittedBlock(ctx context.Context) {
	// the source of truth for the last committed block is persistence
	lastCommittedBlock, err := s.persistence.GetLastBlock()
	if err != nil {
		s.logger.Error(err.Error())
		return
	}

	if lastCommittedBlock != nil {
		// passing nil on purpose, see spec
		err := s.validateWithConsensusAlgos(ctx, nil, lastCommittedBlock)
		if err != nil {
			s.logger.Error(err.Error())
			return
		}
	}
}

func (s *service) HandleBlockAvailabilityRequest(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	err := s.sourceHandleBlockAvailabilityRequest(ctx, input.Message)
	return nil, err
}

func (s *service) HandleBlockAvailabilityResponse(ctx context.Context, input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	if s.extSync != nil {
		s.extSync.HandleBlockAvailabilityResponse(ctx, input)
	}
	return nil, nil
}

func (s *service) HandleBlockSyncRequest(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	err := s.sourceHandleBlockSyncRequest(ctx, input.Message)
	return nil, err
}

func (s *service) HandleBlockSyncResponse(ctx context.Context, input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	if s.extSync != nil {
		s.extSync.HandleBlockSyncResponse(ctx, input)
	}
	return nil, nil
}

// how to check if a block already exists: https://github.com/orbs-network/orbs-spec/issues/50
func (s *service) validateBlockDoesNotExist(txBlockHeader *protocol.TransactionsBlockHeader, rsBlockHeader *protocol.ResultsBlockHeader, lastCommittedBlock *protocol.BlockPairContainer) (bool, error) {
	currentBlockHeight := getBlockHeight(lastCommittedBlock)
	attemptedBlockHeight := txBlockHeader.BlockHeight()

	if attemptedBlockHeight < currentBlockHeight {
		// we can't check for fork because we don't have the tx header of the old block easily accessible
		errorMessage := "block already in storage, skipping"
		s.logger.Info(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight))
		return false, errors.New(errorMessage)
	} else if attemptedBlockHeight == currentBlockHeight {
		// we can check for fork because we do have the tx header of the old block easily accessible
		if txBlockHeader.Timestamp() != getBlockTimestamp(lastCommittedBlock) {
			errorMessage := "FORK!! block already in storage, timestamp mismatch"
			// fork found! this is a major error we must report to logs
			s.logger.Error(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight), log.Stringable("new-block", txBlockHeader), log.Stringable("existing-block", lastCommittedBlock.TransactionsBlock.Header))
			return false, errors.New(errorMessage)
		} else if !txBlockHeader.Equal(lastCommittedBlock.TransactionsBlock.Header) {
			errorMessage := "FORK!! block already in storage, transaction block header mismatch"
			// fork found! this is a major error we must report to logs
			s.logger.Error(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight), log.Stringable("new-block", txBlockHeader), log.Stringable("existing-block", lastCommittedBlock.TransactionsBlock.Header))
			return false, errors.New(errorMessage)
		} else if !rsBlockHeader.Equal(lastCommittedBlock.ResultsBlock.Header) {
			errorMessage := "FORK!! block already in storage, results block header mismatch"
			// fork found! this is a major error we must report to logs
			s.logger.Error(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight), log.Stringable("new-block", rsBlockHeader), log.Stringable("existing-block", lastCommittedBlock.ResultsBlock.Header))
			return false, errors.New(errorMessage)
		}

		s.logger.Info("block already in storage, skipping", log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight))
		return false, nil
	}

	return true, nil
}

func (s *service) validateBlockHeight(blockPair *protocol.BlockPairContainer, lastCommittedBlock *protocol.BlockPairContainer) error {
	expectedBlockHeight := getBlockHeight(lastCommittedBlock) + 1

	txBlockHeader := blockPair.TransactionsBlock.Header
	rsBlockHeader := blockPair.ResultsBlock.Header

	if txBlockHeader.BlockHeight() != expectedBlockHeight {
		return fmt.Errorf("block height is %d, expected %d", txBlockHeader.BlockHeight(), expectedBlockHeight)
	}

	if rsBlockHeader.BlockHeight() != expectedBlockHeight {
		return fmt.Errorf("block height is %d, expected %d", rsBlockHeader.BlockHeight(), expectedBlockHeight)
	}

	return nil
}

func (s *service) validateProtocolVersion(blockPair *protocol.BlockPairContainer) error {
	txBlockHeader := blockPair.TransactionsBlock.Header
	rsBlockHeader := blockPair.ResultsBlock.Header

	// FIXME we may be logging twice, this should be fixed when handling the logging structured errors in logger issue
	if !txBlockHeader.ProtocolVersion().Equal(ProtocolVersion) {
		errorMessage := "protocol version mismatch in transactions block header"
		s.logger.Error(errorMessage, log.Stringable("expected", ProtocolVersion), log.Stringable("received", txBlockHeader.ProtocolVersion()), log.BlockHeight(txBlockHeader.BlockHeight()))
		return fmt.Errorf(errorMessage)
	}

	if !rsBlockHeader.ProtocolVersion().Equal(ProtocolVersion) {
		errorMessage := "protocol version mismatch in results block header"
		s.logger.Error(errorMessage, log.Stringable("expected", ProtocolVersion), log.Stringable("received", rsBlockHeader.ProtocolVersion()), log.BlockHeight(txBlockHeader.BlockHeight()))
		return fmt.Errorf(errorMessage)
	}

	return nil
}

// TODO: this should not be called directly from CommitBlock, it should be called from a long living goroutine that continuously syncs the state storage
func (s *service) syncBlockToStateStorage(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) error {
	_, err := s.stateStorage.CommitStateDiff(ctx, &services.CommitStateDiffInput{
		ResultsBlockHeader: committedBlockPair.ResultsBlock.Header,
		ContractStateDiffs: committedBlockPair.ResultsBlock.ContractStateDiffs,
	})
	return err
}

// TODO: this should not be called directly from CommitBlock, it should be called from a long living goroutine that continuously syncs the state storage
func (s *service) syncBlockToTxPool(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) error {
	_, err := s.txPool.CommitTransactionReceipts(ctx, &services.CommitTransactionReceiptsInput{
		ResultsBlockHeader:       committedBlockPair.ResultsBlock.Header,
		TransactionReceipts:      committedBlockPair.ResultsBlock.TransactionReceipts,
		LastCommittedBlockHeight: committedBlockPair.ResultsBlock.Header.BlockHeight(),
	})
	return err
}

func (s *service) validateWithConsensusAlgos(
	ctx context.Context,
	prevBlockPair *protocol.BlockPairContainer,
	lastCommittedBlockPair *protocol.BlockPairContainer) error {

	return s.validateWithConsensusAlgosWithMode(ctx, prevBlockPair, lastCommittedBlockPair, handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY)
}

func (s *service) validateWithConsensusAlgosWithMode(
	ctx context.Context,
	prevBlockPair *protocol.BlockPairContainer,
	lastCommittedBlockPair *protocol.BlockPairContainer,
	mode handlers.HandleBlockConsensusMode) error {

	for _, handler := range s.consensusBlocksHandlers {
		_, err := handler.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   mode,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              lastCommittedBlockPair,
			PrevCommittedBlockPair: prevBlockPair,
		})

		// one of the consensus algos has validated the block, this means it's a valid block
		if err == nil {
			return nil
		}
	}

	return errors.Errorf("all consensus %d algos refused to validate the block", len(s.consensusBlocksHandlers))
}

// Returns a slice of blocks containing first and last
// TODO support paging
func (s *service) GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight, err error) {
	return s.persistence.GetBlocks(first, last)
}
