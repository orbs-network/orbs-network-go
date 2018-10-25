package blockstorage

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/bloom"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	blockSync "github.com/orbs-network/orbs-network-go/services/blockstorage/sync"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"sync"
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

	lastCommittedBlock *protocol.BlockPairContainer
	lastBlockLock      *sync.RWMutex

	blockSync *blockSync.BlockSync

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
	storage := &service{
		persistence:   persistence,
		stateStorage:  stateStorage,
		gossip:        gossip,
		txPool:        txPool,
		logger:        logger.WithTags(LogTag),
		config:        config,
		lastBlockLock: &sync.RWMutex{},
		metrics:       newMetrics(metricFactory),
	}

	lastBlock, err := persistence.GetLastBlock()

	if err != nil {
		logger.Error("could not update last block from persistence", log.Error(err))
	}

	if lastBlock != nil {
		storage.updateLastCommittedBlock(lastBlock)
	}

	gossip.RegisterBlockSyncHandler(storage)
	storage.blockSync = blockSync.NewBlockSync(ctx, config, gossip, storage, logger)

	return storage
}

func (s *service) CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	txBlockHeader := input.BlockPair.TransactionsBlock.Header
	s.logger.Info("Trying to commit a block", log.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.validateProtocolVersion(input.BlockPair); err != nil {
		return nil, err
	}

	// TODO there might be a non-idiomatic pattern here, but the commit block output is an empty struct, if that changes this should be refactored
	if ok, err := s.validateBlockDoesNotExist(txBlockHeader); err != nil || !ok {
		return nil, err
	}

	if err := s.validateBlockHeight(input.BlockPair); err != nil {
		return nil, err
	}

	if err := s.persistence.WriteBlock(input.BlockPair); err != nil {
		return nil, err
	}

	s.updateLastCommittedBlock(input.BlockPair)
	s.blockSync.HandleBlockCommitted()

	s.logger.Info("committed a block", log.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.syncBlockToStateStorage(ctx, input.BlockPair); err != nil {
		// TODO: since the intra-node sync flow is self healing, we should not fail the entire commit if state storage is slow to sync
		s.logger.Error("intra-node sync to state storage failed", log.Error(err))
	}

	if err := s.syncBlockToTxPool(ctx, input.BlockPair); err != nil {
		// TODO: since the intra-node sync flow is self healing, should we fail if pool fails ?
		s.logger.Error("intra-node sync to tx pool failed", log.Error(err))
	}

	return nil, nil
}

func (s *service) updateLastCommittedBlock(block *protocol.BlockPairContainer) {
	s.lastBlockLock.Lock()
	defer s.lastBlockLock.Unlock()

	blockHeight := int64(block.TransactionsBlock.Header.BlockHeight())
	s.metrics.blockHeight.Update(blockHeight)
	s.lastCommittedBlock = block
}

func (s *service) LastCommittedBlockHeight() primitives.BlockHeight {
	s.lastBlockLock.RLock()
	defer s.lastBlockLock.RUnlock()

	if s.lastCommittedBlock == nil {
		return 0
	}
	return s.lastCommittedBlock.TransactionsBlock.Header.BlockHeight()
}

func (s *service) lastCommittedBlockTimestamp() primitives.TimestampNano {
	s.lastBlockLock.RLock()
	defer s.lastBlockLock.RUnlock()

	if s.lastCommittedBlock == nil {
		return 0
	}
	return s.lastCommittedBlock.TransactionsBlock.Header.Timestamp()
}

func (s *service) getLastCommittedBlock() *protocol.BlockPairContainer {
	s.lastBlockLock.RLock()
	defer s.lastBlockLock.RUnlock()

	return s.lastCommittedBlock
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

func (s *service) createEmptyTransactionReceiptResult() *services.GetTransactionReceiptOutput {
	return &services.GetTransactionReceiptOutput{
		TransactionReceipt: nil,
		BlockHeight:        s.LastCommittedBlockHeight(),
		BlockTimestamp:     s.lastCommittedBlockTimestamp(),
	}
}

func (s *service) GetTransactionReceipt(ctx context.Context, input *services.GetTransactionReceiptInput) (*services.GetTransactionReceiptOutput, error) {
	searchRules := adapter.BlockSearchRules{
		EndGraceNano:          s.config.BlockTransactionReceiptQueryGraceEnd().Nanoseconds(),
		StartGraceNano:        s.config.BlockTransactionReceiptQueryGraceStart().Nanoseconds(),
		TransactionExpireNano: s.config.BlockTransactionReceiptQueryExpirationWindow().Nanoseconds(),
	}
	blocksToSearch := s.persistence.GetReceiptRelevantBlocks(input.TransactionTimestamp, searchRules)
	if blocksToSearch == nil {
		return s.createEmptyTransactionReceiptResult(), errors.Errorf("failed to search for blocks on tx timestamp of %d, hash %s", input.TransactionTimestamp, input.Txhash)
	}

	if len(blocksToSearch) == 0 {
		return s.createEmptyTransactionReceiptResult(), nil
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

	return s.createEmptyTransactionReceiptResult(), nil
}

func (s *service) GetLastCommittedBlockHeight(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
	b := s.getLastCommittedBlock()
	if b == nil {
		return &services.GetLastCommittedBlockHeightOutput{
			LastCommittedBlockHeight:    0,
			LastCommittedBlockTimestamp: 0,
		}, nil
	}
	return &services.GetLastCommittedBlockHeightOutput{
		LastCommittedBlockHeight:    b.TransactionsBlock.Header.BlockHeight(),
		LastCommittedBlockTimestamp: b.TransactionsBlock.Header.Timestamp(),
	}, nil
}

// FIXME implement all block checks
func (s *service) ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	if protocolVersionError := s.validateProtocolVersion(input.BlockPair); protocolVersionError != nil {
		return nil, protocolVersionError
	}

	if blockHeightError := s.validateBlockHeight(input.BlockPair); blockHeightError != nil {
		return nil, blockHeightError
	}

	if err := s.validateWithConsensusAlgosWithMode(
		ctx,
		s.lastCommittedBlock,
		input.BlockPair,
		handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE); err != nil {

		s.logger.Error("intra-node sync to consensus algo failed", log.Error(err))
	}

	return &services.ValidateBlockForCommitOutput{}, nil
}

func (s *service) RegisterConsensusBlocksHandler(handler handlers.ConsensusBlocksHandler) {
	s.consensusBlocksHandlers = append(s.consensusBlocksHandlers, handler)

	// update the consensus algo about the latest block we have (for its initialization)
	// TODO: should this be under mutex since it reads s.lastCommittedBlock
	s.UpdateConsensusAlgosAboutLatestCommittedBlock(context.TODO()) // TODO: (talkol) not sure if we should create a new context here or pass to RegisterConsensusBlocksHandler in code generation
}

func (s *service) UpdateConsensusAlgosAboutLatestCommittedBlock(ctx context.Context) {
	lastCommitted := s.getLastCommittedBlock()

	if lastCommitted != nil {
		// passing nil on purpose, see spec
		err := s.validateWithConsensusAlgos(ctx, nil, lastCommitted)
		if err != nil {
			s.logger.Error(err.Error())
		}
	}
}

func (s *service) HandleBlockAvailabilityRequest(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	err := s.sourceHandleBlockAvailabilityRequest(ctx, input.Message)
	return nil, err
}

func (s *service) HandleBlockAvailabilityResponse(ctx context.Context, input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	if s.blockSync != nil {
		s.blockSync.HandleBlockAvailabilityResponse(ctx, input)
	}
	return nil, nil
}

func (s *service) HandleBlockSyncRequest(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	err := s.sourceHandleBlockSyncRequest(ctx, input.Message)
	return nil, err
}

func (s *service) HandleBlockSyncResponse(ctx context.Context, input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	if s.blockSync != nil {
		s.blockSync.HandleBlockSyncResponse(ctx, input)
	}
	return nil, nil
}

// how to check if a block already exists: https://github.com/orbs-network/orbs-spec/issues/50
func (s *service) validateBlockDoesNotExist(txBlockHeader *protocol.TransactionsBlockHeader) (bool, error) {
	currentBlockHeight := s.LastCommittedBlockHeight()
	attemptedBlockHeight := txBlockHeader.BlockHeight()

	if attemptedBlockHeight < currentBlockHeight {
		// we can't check for fork because we don't have the tx header of the old block easily accessible
		errorMessage := "block already in storage, skipping"
		s.logger.Info(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight))
		return false, errors.New(errorMessage)
	} else if attemptedBlockHeight == currentBlockHeight {
		// we can check for fork because we do have the tx header of the old block easily accessible
		if txBlockHeader.Timestamp() != s.lastCommittedBlockTimestamp() {
			errorMessage := "FORK!! block already in storage, timestamp mismatch"
			// fork found! this is a major error we must report to logs
			s.logger.Error(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight))
			return false, errors.New(errorMessage)
		} else if !txBlockHeader.Equal(s.lastCommittedBlock.TransactionsBlock.Header) {
			errorMessage := "FORK!! block already in storage, transaction block header mismatch"
			// fork found! this is a major error we must report to logs
			s.logger.Error(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight))
			return false, errors.New(errorMessage)
		}

		s.logger.Info("block already in storage, skipping", log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight))
		return false, nil
	}

	return true, nil
}

func (s *service) validateBlockHeight(blockPair *protocol.BlockPairContainer) error {
	expectedBlockHeight := s.LastCommittedBlockHeight() + 1

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
// TODO support chunking
func (s *service) GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight) {
	// FIXME use more efficient way to slice blocks

	allBlocks := s.persistence.ReadAllBlocks()
	allBlocksLength := primitives.BlockHeight(len(allBlocks))

	s.logger.Info("Reading all blocks", log.Stringable("blocks-total", allBlocksLength))

	firstAvailableBlockHeight = first

	// FIXME what does it even mean
	if firstAvailableBlockHeight > allBlocksLength {
		return blocks, firstAvailableBlockHeight, firstAvailableBlockHeight
	}

	lastAvailableBlockHeight = last
	if allBlocksLength < last {
		lastAvailableBlockHeight = allBlocksLength
	}

	for i := first - 1; i < lastAvailableBlockHeight; i++ {
		s.logger.Info("Retrieving block", log.BlockHeight(i), log.Stringable("blocks-total", i))
		blocks = append(blocks, allBlocks[i])
	}

	return blocks, firstAvailableBlockHeight, lastAvailableBlockHeight
}
