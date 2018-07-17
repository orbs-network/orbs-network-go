package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/syndtr/goleveldb/leveldb"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
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

func (bp *levelDbBlockPersistence) retrieveByPrefix(prefix string) (results [][]byte) {
	iter := bp.db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)

	for iter.Next() {
		println("Retrieving key", string(iter.Key()))
		results = append(results, copyByteArray(iter.Value()))
	}

	iter.Release()

	return results
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

func copyArrayOfByteArrays(data [][]byte) [][]byte {
	result := make([][]byte, len(data))
	copy(result, data)

	return result
}

func formatInt(i int) string {
	return strconv.FormatInt(int64(i), 10)
}

func (bp *levelDbBlockPersistence) putTxBlock(txBlock *protocol.TransactionsBlockContainer) (errors []error, keys []string) {
	blockHeight := strconv.FormatUint(uint64(txBlock.Header.BlockHeight()), 10)

	txBlockHeaderKey := TX_BLOCK_HEADER + blockHeight
	txBlockProofKey := TX_BLOCK_PROOF + blockHeight
	txBlockMetadataKey := TX_BLOCK_METADATA + blockHeight

	txBlockHeaderError := bp.put(txBlockHeaderKey, txBlock.Header.Raw())
	txBlockProofError := bp.put(txBlockProofKey, txBlock.BlockProof.Raw())
	txBlockMetadataError := bp.put(txBlockMetadataKey, txBlock.Metadata.Raw())

	keys = append(keys, txBlockHeaderKey, txBlockProofKey, txBlockMetadataKey)
	errors = append(errors, txBlockHeaderError, txBlockProofError, txBlockMetadataError)

	for i, tx := range txBlock.SignedTransactions {
		txBlockSignedTransactionKey := TX_BLOCK_SIGNED_TRANSACTION + blockHeight + "-" + formatInt(i)
		txBlockSignedTransactionError := bp.put(txBlockSignedTransactionKey, tx.Raw())

		keys = append(keys, txBlockSignedTransactionKey)
		errors = append(errors, txBlockSignedTransactionError)
	}

	return errors, keys
}

func (bp *levelDbBlockPersistence) putResultsBlock(rsBlock *protocol.ResultsBlockContainer) (errors []error, keys []string) {
	blockHeight := strconv.FormatUint(uint64(rsBlock.Header.BlockHeight()), 10)

	rsBlockHeaderKey := RS_BLOCK_HEADER + blockHeight
	rsBlockProofKey := RS_BLOCK_PROOF + blockHeight

	rsBlockHeaderError := bp.put(rsBlockHeaderKey, rsBlock.Header.Raw())
	rsBlockProofError := bp.put(rsBlockProofKey, rsBlock.BlockProof.Raw())

	keys = append(keys, rsBlockHeaderKey, rsBlockProofKey)
	errors = append(errors, rsBlockHeaderError, rsBlockProofError)


	for i, sd := range rsBlock.ContractStateDiffs {
		rsBlockContractStatesDiffsKey := RS_BLOCK_CONTRACT_STATE_DIFFS + blockHeight + "-" + formatInt(i)
		rsBlockContractStatesDiffsError := bp.put(rsBlockContractStatesDiffsKey, sd.Raw())

		keys = append(keys, rsBlockContractStatesDiffsKey)
		errors = append(errors, rsBlockContractStatesDiffsError)
	}

	for i, tr := range rsBlock.TransactionReceipts {
		rsBlockTransactionReceiptsKey := RS_BLOCK_TRANSACTION_RECEIPTS + blockHeight + "-" + formatInt(i)
		rsBlockTransactionReceiptsError := bp.put(rsBlockTransactionReceiptsKey, tr.Raw())

		keys = append(keys, rsBlockTransactionReceiptsKey)
		errors = append(errors, rsBlockTransactionReceiptsError)
	}

	return errors, keys
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

func constructTxBlockFromStorage(txBlockHeaderRaw []byte, txBlockProofRaw []byte, txBlockMetadataRaw []byte,
	txBlockSignedTransactionsRaw [][]byte) *protocol.TransactionsBlockContainer {
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

	return transactionsBlock
}

func constructResultsBlockFromStorage(rsBlockHeaderRaw []byte, rsBlockProofRaw []byte, rsBlockStateDiffsRaw [][]byte, rsTransactionReceiptsRaw [][]byte) *protocol.ResultsBlockContainer {
	var transactionReceipts []*protocol.TransactionReceipt
	var stateDiffs []*protocol.ContractStateDiff

	for _, trRaw := range rsTransactionReceiptsRaw {
		transactionReceipts = append(transactionReceipts, protocol.TransactionReceiptReader(trRaw))
	}

	for _, sdRaw := range rsBlockStateDiffsRaw {
		stateDiffs = append(stateDiffs, protocol.ContractStateDiffReader(sdRaw))
	}

	resultsBlock := &protocol.ResultsBlockContainer{
		Header: protocol.ResultsBlockHeaderReader(rsBlockHeaderRaw),
		BlockProof: protocol.ResultsBlockProofReader(rsBlockProofRaw),
		ContractStateDiffs: stateDiffs,
		TransactionReceipts: transactionReceipts,
	}

	return resultsBlock
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