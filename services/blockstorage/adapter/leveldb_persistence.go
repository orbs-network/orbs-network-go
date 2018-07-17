package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/syndtr/goleveldb/leveldb"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
	"strings"
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

// FIXME should I handle errors?
func (bp *levelDbBlockPersistence) retrieve(key string) []byte {
	fmt.Printf("Retrieving key %v\n", key)

	result, _ := bp.db.Get([]byte(key), nil)
	return result
}


func (bp *levelDbBlockPersistence) revert(key string) error {
	fmt.Println("Removing key", key)

	return bp.db.Delete([]byte(key), nil)
}

func copyByteArray(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)

	return result
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
		txBlockSignedTransactionKey := "transaction-block-signed-transaction-" + blockHeight + "-" + strconv.FormatInt(int64(i), 10)
		txBlockSignedTransactionError := bp.put(txBlockSignedTransactionKey, tx.Raw())

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

func constructBlockFromStorage(txBlockHeaderRaw []byte, txBlockProofRaw []byte, txBlockMetadataRaw []byte,
	txBlockSignedTransactionsRaw [][]byte) *protocol.BlockPairContainer {
	var signedTransactions []*protocol.SignedTransaction

	for _, txRaw := range txBlockSignedTransactionsRaw {
		signedTransactions = append(signedTransactions, protocol.SignedTransactionReader(txRaw))
	}

	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header: protocol.TransactionsBlockHeaderReader(txBlockHeaderRaw),
		BlockProof: protocol.TransactionsBlockProofReader(txBlockProofRaw),
		Metadata: protocol.TransactionsBlockMetadataReader(txBlockMetadataRaw),
		SignedTransactions: signedTransactions,
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
		tokenizedKey := strings.Split(key, "-")
		blockHeightAsString := tokenizedKey[len(tokenizedKey) - 1]

		txBlockHeaderRaw := copyByteArray(iter.Value())
		txBlockProofRaw := copyByteArray(bp.retrieve("transaction-block-proof-" + blockHeightAsString))
		txBlockMetadataRaw := copyByteArray(bp.retrieve("transaction-block-metadata-" + blockHeightAsString))

		var txSignedTransactionsRaw [][]byte

		txIter := bp.db.NewIterator(util.BytesPrefix([]byte("transaction-block-signed-transaction-" + blockHeightAsString + "-")), nil)

		for txIter.Next() {
			println("Retrieving key", string(txIter.Key()))
			txSignedTransactionsRaw = append(txSignedTransactionsRaw, copyByteArray(txIter.Value()))
		}

		txIter.Release()

		results = append(results, constructBlockFromStorage(txBlockHeaderRaw, txBlockProofRaw, txBlockMetadataRaw, txSignedTransactionsRaw))
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