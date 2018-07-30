package adapter

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	LAST_BLOCK_HEIGHT = "last-block-height"

	TX_BLOCK_HEADER             = "transaction-block-header-"
	TX_BLOCK_PROOF              = "transaction-block-proof-"
	TX_BLOCK_METADATA           = "transaction-block-metadata-"
	TX_BLOCK_SIGNED_TRANSACTION = "transaction-block-signed-transaction-"

	RS_BLOCK_HEADER               = "results-block-header-"
	RS_BLOCK_PROOF                = "results-block-proof-"
	RS_BLOCK_CONTRACT_STATE_DIFFS = "results-block-contract-state-diffs-"
	RS_BLOCK_TRANSACTION_RECEIPTS = "results-block-transaction-receipts-"
)

type Config interface {
}

type levelDbBlockPersistence struct {
	blockWritten chan bool
	blockPairs   []*protocol.BlockPairContainer
	config       Config
	reporting    instrumentation.BasicLogger
	db           *leveldb.DB
}

type config struct {
	name string
}

func (c *config) NodeId() string {
	return c.name
}

func NewLevelDbBlockPersistenceConfig(name string) Config {
	return &config{name: name}
}

func NewLevelDbBlockPersistence(config Config) BlockPersistence {
	db, err := leveldb.OpenFile("/tmp/db", nil)

	if err != nil {
		instrumentation.GetLogger(instrumentation.String("component", "persistence")).Error("Could not instantiate leveldb", instrumentation.Error(err))
		panic("Could not instantiate leveldb")
	}

	return &levelDbBlockPersistence{
		config:       config,
		blockWritten: make(chan bool, 10),
		db:           db,
		reporting:    instrumentation.GetLogger(),
	}
}

func (bp *levelDbBlockPersistence) WithLogger(reporting instrumentation.BasicLogger) BlockPersistence {
	bp.reporting = reporting
	return bp
}

func (bp *levelDbBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) {
	var errors []error
	var keys []string

	if !basicValidation(blockPair) {
		bp.reporting.Info("Block is invalid", instrumentation.Stringable("txBlockHeader", blockPair.TransactionsBlock.Header))
		return
	}

	lastBlockHeight, lastBlockHeightRetrivalError := bp.loadLastBlockHeight()
	errors = append(errors, lastBlockHeightRetrivalError)

	txErrors, txKeys := bp.putTxBlock(blockPair.TransactionsBlock)
	rsErrors, rsKeys := bp.putResultsBlock(blockPair.ResultsBlock)

	newBlockHeight := blockPair.TransactionsBlock.Header.BlockHeight()
	blockHeightSaveError := bp.saveLastBlockHeight(newBlockHeight)

	errors = append(errors, blockHeightSaveError)

	errors = append(errors, txErrors...)
	errors = append(errors, rsErrors...)

	keys = append(keys, txKeys...)
	keys = append(keys, rsKeys...)

	if anyErrors(errors) {
		bp.saveLastBlockHeight(lastBlockHeight)
		bp.reporting.Error("Failed to write block", instrumentation.BlockHeight(newBlockHeight))

		for _, key := range keys {
			bp.revert(key)
		}

		bp.blockWritten <- false
		return
	}

	bp.blockWritten <- true
}

func (bp *levelDbBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	var results []*protocol.BlockPairContainer

	lastBlockHeight, _ := bp.loadLastBlockHeight()

	for i := uint64(1); i <= lastBlockHeight.KeyForMap(); i++ {
		currentBlockHeight := primitives.BlockHeight(i)
		currentTxBlock, _ := bp.GetTransactionsBlock(currentBlockHeight)
		currentRsBlock, _ := bp.GetResultsBlock(currentBlockHeight)

		results = append(results, &protocol.BlockPairContainer{
			TransactionsBlock: currentTxBlock,
			ResultsBlock:      currentRsBlock,
		})
	}

	return results
}

func basicValidation(blockPair *protocol.BlockPairContainer) bool {
	var validations []bool

	validations = append(validations, blockPair.TransactionsBlock.Header.IsValid(), blockPair.TransactionsBlock.BlockProof.IsValid(), blockPair.TransactionsBlock.Metadata.IsValid())

	for _, tx := range blockPair.TransactionsBlock.SignedTransactions {
		validations = append(validations, tx.IsValid())
	}

	return anyConditions(validations)
}

func (bp *levelDbBlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	blockHeightAsString := height.String()

	txBlockHeaderRaw := bp.retrieve(TX_BLOCK_HEADER + blockHeightAsString)
	txBlockProofRaw := bp.retrieve(TX_BLOCK_PROOF + blockHeightAsString)
	txBlockMetadataRaw := bp.retrieve(TX_BLOCK_METADATA + blockHeightAsString)

	txSignedTransactionsRaw := bp.retrieveByPrefix(TX_BLOCK_SIGNED_TRANSACTION + blockHeightAsString + "-")

	bp.reporting.Info("Retrieved transactions block from storage", instrumentation.BlockHeight(height))

	return constructTxBlockFromStorage(txBlockHeaderRaw, txBlockProofRaw, txBlockMetadataRaw, txSignedTransactionsRaw), nil
}

func (bp *levelDbBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	blockHeightAsString := height.String()

	rsBlockHeaderRaw := bp.retrieve(RS_BLOCK_HEADER + blockHeightAsString)
	rsBlockProofRaw := bp.retrieve(RS_BLOCK_PROOF + blockHeightAsString)

	rsTransactionReceipts := bp.retrieveByPrefix(RS_BLOCK_TRANSACTION_RECEIPTS + blockHeightAsString + "-")
	rsStateDiffs := bp.retrieveByPrefix(RS_BLOCK_CONTRACT_STATE_DIFFS + blockHeightAsString + "-")

	bp.reporting.Info("Retrieved results block from storage", instrumentation.BlockHeight(height))

	return constructResultsBlockFromStorage(rsBlockHeaderRaw, rsBlockProofRaw, rsStateDiffs, rsTransactionReceipts), nil
}
