package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/syndtr/goleveldb/leveldb"
	"strconv"
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
		fmt.Println("Could not instantiate leveldb", err)
		panic("Could not instantiate leveldb")
	}

	return &levelDbBlockPersistence{
		config:       config,
		blockWritten: make(chan bool, 10),
		db:           db,
	}
}

func (bp *levelDbBlockPersistence) loadLastBlockHeight() (primitives.BlockHeight, error) {
	val, err := bp.db.Get([]byte(LAST_BLOCK_HEIGHT), nil)

	if err != nil {
		return 0, nil
	}

	result, err := strconv.ParseUint(string(val), 16, 64)
	return primitives.BlockHeight(result), err
}

func (bp *levelDbBlockPersistence) saveLastBlockHeight(height primitives.BlockHeight) error {
	return bp.db.Put([]byte(LAST_BLOCK_HEIGHT), []byte(height.String()), nil)
}

func (bp *levelDbBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) {
	var errors []error
	var keys []string

	if !basicValidation(blockPair) {
		fmt.Println("Block is invalid")
		return
	}

	lastBlockHeight, lastBlockHeightRetrivalError := bp.loadLastBlockHeight()
	errors = append(errors, lastBlockHeightRetrivalError)

	txErrors, txKeys := bp.putTxBlock(blockPair.TransactionsBlock)
	rsErrors, rsKeys := bp.putResultsBlock(blockPair.ResultsBlock)

	blockHeightSaveError := bp.saveLastBlockHeight(blockPair.TransactionsBlock.Header.BlockHeight())

	errors = append(errors, blockHeightSaveError)

	errors = append(errors, txErrors...)
	errors = append(errors, rsErrors...)

	keys = append(keys, txKeys...)
	keys = append(keys, rsKeys...)

	if anyErrors(errors) {
		fmt.Println("Failed to write block")

		bp.saveLastBlockHeight(lastBlockHeight)

		for _, key := range keys {
			bp.revert(key)
		}

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

	txBlockHeaderRaw := copyByteArray(bp.retrieve(TX_BLOCK_HEADER + blockHeightAsString))
	txBlockProofRaw := copyByteArray(bp.retrieve(TX_BLOCK_PROOF + blockHeightAsString))
	txBlockMetadataRaw := copyByteArray(bp.retrieve(TX_BLOCK_METADATA + blockHeightAsString))

	txSignedTransactionsRaw := bp.retrieveByPrefix(TX_BLOCK_SIGNED_TRANSACTION + blockHeightAsString + "-")

	return constructTxBlockFromStorage(txBlockHeaderRaw, txBlockProofRaw, txBlockMetadataRaw, txSignedTransactionsRaw), nil
}

func (bp *levelDbBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	blockHeightAsString := height.String()

	rsBlockHeaderRaw := copyByteArray(bp.retrieve(RS_BLOCK_HEADER + blockHeightAsString))
	rsBlockProofRaw := copyByteArray(bp.retrieve(RS_BLOCK_PROOF + blockHeightAsString))

	rsTransactionReceipts := copyArrayOfByteArrays(bp.retrieveByPrefix(RS_BLOCK_TRANSACTION_RECEIPTS + blockHeightAsString + "-"))
	rsStateDiffs := copyArrayOfByteArrays(bp.retrieveByPrefix(RS_BLOCK_CONTRACT_STATE_DIFFS + blockHeightAsString + "-"))

	return constructResultsBlockFromStorage(rsBlockHeaderRaw, rsBlockProofRaw, rsStateDiffs, rsTransactionReceipts), nil
}
