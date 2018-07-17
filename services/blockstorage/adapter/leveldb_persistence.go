package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/syndtr/goleveldb/leveldb"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
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

func (bp *levelDbBlockPersistence) put(key string, value []byte) error {
	fmt.Printf("Writing key %v, value %v\n", key, value)

	return bp.db.Put([]byte(key), value, nil)
}

func (bp *levelDbBlockPersistence) revert(key string) error {
	fmt.Println("Removing key", key)

	return bp.db.Delete([]byte(key), nil)
}

func (bp *levelDbBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) {
	var errors []error
	var keys []string

	if !basicValidation(blockPair) {
		fmt.Println("Block is invalid")
		return
	}

	blockHeight := strconv.FormatUint(uint64(blockPair.TransactionsBlock.Header.BlockHeight()), 10)

	txBlockHeaderKey := "transaction-block-header-" + blockHeight
	txBlockProofKey := "transaction-block-proof-" + blockHeight
	txBlockMetadataKey := "transaction-block-metadata-" + blockHeight

	txBlockHeaderError := bp.put(txBlockHeaderKey, blockPair.TransactionsBlock.Header.Raw())
	txBlockProofError := bp.put(txBlockProofKey, blockPair.TransactionsBlock.BlockProof.Raw())
	txBlockMetadataError := bp.put(txBlockMetadataKey, blockPair.TransactionsBlock.Metadata.Raw())

	for i, tx := range blockPair.TransactionsBlock.SignedTransactions {
		txBlockSignedTransactionKey := "transaction-block-proof-" + blockHeight + "-" + strconv.FormatInt(int64(i), 10)
		txBlockSignedTransactionError := bp.put(txBlockProofKey, tx.Raw())

		keys = append(keys, txBlockSignedTransactionKey)
		errors = append(errors, txBlockSignedTransactionError)
	}

	keys = append(keys, txBlockHeaderKey, txBlockProofKey, txBlockMetadataKey)
	errors = append(errors, txBlockHeaderError, txBlockProofError, txBlockMetadataError)

	if anyErrors(errors) {
		fmt.Println("Failed to write block")

		for _, key := range keys {
			bp.revert(key)
		}

		return
	}

	bp.blockWritten <- true
}

func constructBlockFromStorage(data []byte) *protocol.BlockPairContainer {
	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header: protocol.TransactionsBlockHeaderReader(data),
	}

	resultsBlock := &protocol.ResultsBlockContainer{}

	container := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock: resultsBlock,
	}

	return container
}

func (bp *levelDbBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	var results []*protocol.BlockPairContainer

	iter := bp.db.NewIterator(util.BytesPrefix([]byte("transaction-block-header-")), nil)

	for iter.Next()  {
		key := string(iter.Key())
		data := make([]byte, len(iter.Value()))
		copy(data, iter.Value())

		fmt.Printf("Retrieving key %v, value %v\n", key, data)

		results = append(results, constructBlockFromStorage(data))
	}
	iter.Release()
	_ = iter.Error()

	return results
}

func anyErrors(errors []error) bool {
	for _, error := range errors {
		if error != nil {
			fmt.Println("Found error", errors)
			return true
		}
	}

	return false
}

func anyConditions(bools []bool) bool {
	for _, val := range bools {
		if val == false {
			return false
		}
	}

	return true
}

func basicValidation(blockPair *protocol.BlockPairContainer) bool {
	var validations []bool

	validations = append(validations, blockPair.TransactionsBlock.Header.IsValid(), blockPair.TransactionsBlock.BlockProof.IsValid(), blockPair.TransactionsBlock.Metadata.IsValid())

	for _, tx:= range blockPair.TransactionsBlock.SignedTransactions {
		validations = append(validations, tx.IsValid())
	}

	return anyConditions(validations)
}