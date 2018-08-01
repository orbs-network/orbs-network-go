package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	LAST_BLOCK_HEIGHT    = "last-block-height"
	LAST_BLOCK_TIMESTAMP = "last-block-timestamp"

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

func (bp *levelDbBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) error {
	var errors []error
	var keys []string

	if !basicValidation(blockPair) {
		//FIXME: handle errors
		bp.reporting.Info("Block is invalid", instrumentation.Stringable("txBlockHeader", blockPair.TransactionsBlock.Header))
		return fmt.Errorf("block is invalid")
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

	if hasErrors, firstError := anyErrors(errors...); hasErrors {
		bp.saveLastBlockHeight(lastBlockHeight)
		bp.reporting.Error("Failed to write block", instrumentation.BlockHeight(newBlockHeight), instrumentation.Error(firstError))

		for _, key := range keys {
			bp.revert(key)
		}

		bp.blockWritten <- false
		return nil
	}
	bp.blockWritten <- true

	return nil
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

func (bp *levelDbBlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	lastBlockHeight, err := bp.loadLastBlockHeight()

	if err != nil {
		return nil, err
	}

	if height > lastBlockHeight {
		return nil, fmt.Errorf("transactions block with this height does not exist yet")
	}

	return bp.loadTransactionsBlock(height)
}

func (bp *levelDbBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	lastBlockHeight, err := bp.loadLastBlockHeight()

	if err != nil {
		return nil, err
	}

	if height > lastBlockHeight {
		return nil, fmt.Errorf("results block with this height does not exist yet")
	}

	return bp.loadResultsBlock(height)
}

func (bp *levelDbBlockPersistence) GetLastBlockDetails() (primitives.BlockHeight, primitives.TimestampNano) {
	height, heightError := bp.loadLastBlockHeight()
	timestamp, timestampError := bp.loadLastBlockTimestamp()

	if hasErrors, firstError := anyErrors(heightError, timestampError); hasErrors {
		bp.reporting.Error("failed to retrieve last block details", instrumentation.Error(firstError))
	}

	return height, timestamp
}
