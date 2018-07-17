package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/syndtr/goleveldb/leveldb"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strings"
)

const (
	TX_BLOCK_HEADER = "transaction-block-header-"
	TX_BLOCK_PROOF = "transaction-block-proof-"
	TX_BLOCK_METADATA = "transaction-block-metadata-"
	TX_BLOCK_SIGNED_TRANSACTION = "transaction-block-signed-transaction-"

	RS_BLOCK_HEADER = "results-block-header-"
	RS_BLOCK_PROOF = "results-block-proof-"
	RS_BLOCK_CONTRACT_STATE_DIFFS = "results-block-contract-state-diffs-"
	RS_BLOCK_TRANSACTION_RECEIPTS = "results-block-transaction-receipts-"
)


type Config interface {
	NodeId() string
}

type levelDbBlockPersistence struct {
	blockWritten chan bool
	blockPairs   []*protocol.BlockPairContainer
	config       Config
	db *leveldb.DB
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

	if err != nil{
		fmt.Println("Could not instantiate leveldb", err)
		panic("Could not instantiate leveldb")
	}

	return &levelDbBlockPersistence{
		config:       config,
		blockWritten: make(chan bool, 10),
		db: db,
	}
}

func (bp *levelDbBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) {
	var errors []error
	var keys []string

	if !basicValidation(blockPair) {
		fmt.Println("Block is invalid")
		return
	}

	txErrors, txKeys := bp.putTxBlock(blockPair.TransactionsBlock)
	rsErrors, rsKeys := bp.putResultsBlock(blockPair.ResultsBlock)

	errors = append(errors, txErrors...)
	errors = append(errors, rsErrors...)

	keys = append(keys, txKeys...)
	keys = append(keys, rsKeys...)

	if anyErrors(errors) {
		fmt.Println("Failed to write block")

		for _, key := range keys {
			bp.revert(key)
		}

		return
	}

	bp.blockWritten <- true
}

func (bp *levelDbBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	var results []*protocol.BlockPairContainer

	iter := bp.db.NewIterator(util.BytesPrefix([]byte(TX_BLOCK_HEADER)), nil)

	for iter.Next()  {
		key := string(iter.Key())
		tokenizedKey := strings.Split(key, "-")
		blockHeightAsString := tokenizedKey[len(tokenizedKey) - 1]

		txBlockHeaderRaw := copyByteArray(iter.Value())
		txBlockProofRaw := copyByteArray(bp.retrieve(TX_BLOCK_PROOF + blockHeightAsString))
		txBlockMetadataRaw := copyByteArray(bp.retrieve(TX_BLOCK_METADATA + blockHeightAsString))

		txSignedTransactionsRaw := bp.retrieveByPrefix(TX_BLOCK_SIGNED_TRANSACTION + blockHeightAsString + "-")

		rsBlockHeaderRaw := copyByteArray(bp.retrieve(RS_BLOCK_HEADER + blockHeightAsString))
		rsBlockProofRaw := copyByteArray(bp.retrieve(RS_BLOCK_PROOF + blockHeightAsString))

		rsTransactionReceipts := copyArrayOfByteArrays(bp.retrieveByPrefix(RS_BLOCK_TRANSACTION_RECEIPTS + blockHeightAsString + "-"))
		rsStateDiffs := copyArrayOfByteArrays(bp.retrieveByPrefix(RS_BLOCK_CONTRACT_STATE_DIFFS + blockHeightAsString + "-"))

		container := &protocol.BlockPairContainer{
			TransactionsBlock: constructTxBlockFromStorage(txBlockHeaderRaw, txBlockProofRaw, txBlockMetadataRaw, txSignedTransactionsRaw),
			ResultsBlock: constructResultsBlockFromStorage(rsBlockHeaderRaw, rsBlockProofRaw, rsStateDiffs, rsTransactionReceipts),
		}

		results = append(results, container)
	}
	iter.Release()
	_ = iter.Error()

	return results
}

func basicValidation(blockPair *protocol.BlockPairContainer) bool {
	var validations []bool

	validations = append(validations, blockPair.TransactionsBlock.Header.IsValid(), blockPair.TransactionsBlock.BlockProof.IsValid(), blockPair.TransactionsBlock.Metadata.IsValid())

	for _, tx:= range blockPair.TransactionsBlock.SignedTransactions {
		validations = append(validations, tx.IsValid())
	}

	return anyConditions(validations)
}