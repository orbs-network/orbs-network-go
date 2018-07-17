package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"strconv"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

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