package adapter

import (
	"strconv"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"bytes"
	"encoding/binary"
)

func copyByteArray(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)

	return result
}

func copyArrayOfByteArrays(data [][]byte) [][]byte {
	result := make([][]byte, len(data))
	for i := range data {
		result[i] = make([]byte, len(data[i]))
		copy(result[i], data[i])
	}

	return result
}

func formatInt(i int) string {
	return strconv.FormatInt(int64(i), 10)
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

func bufferPutKeyValue(buffer *bytes.Buffer, key string, value []byte) {
	bufferPutValue(buffer, []byte(key))
	bufferPutValue(buffer, value)
}

func bufferPutValue(buffer *bytes.Buffer, value []byte) {
	valueLength := uint64(len(value))

	valueLengthAsBytes := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(valueLengthAsBytes, valueLength)

	buffer.Write(valueLengthAsBytes)
	buffer.Write([]byte(value))
}

func bufferReadValue(data []byte, offset int) (value []byte, newOffset int) {
	keyLengthStart := offset
	keyLengthEnd := offset+binary.MaxVarintLen64

	keyLength, _ := binary.ReadUvarint(bytes.NewReader(data[keyLengthStart:keyLengthEnd]))

	keyStart := keyLengthEnd
	keyEnd := keyStart +int(keyLength)

	key := data[keyStart:keyEnd]

	return key, keyEnd
}

func iterateOverKeyValueBuffer(buffer *bytes.Buffer, parseValue func(key string, value []byte)) {
	data := buffer.Bytes()
	offset := 0

	var key []byte
	var value []byte

	for offset < len(data)  {
		key, offset = bufferReadValue(data, offset)
		value, offset = bufferReadValue(data, offset)

		parseValue(string(key), value)
	}
}

func blockAsByteArray(container *protocol.BlockPairContainer) (result []byte) {
	buffer := bytes.NewBuffer([]byte{})

	bufferPutKeyValue(buffer, TX_BLOCK_HEADER, container.TransactionsBlock.Header.Raw())
	bufferPutKeyValue(buffer, TX_BLOCK_PROOF, container.TransactionsBlock.BlockProof.Raw())
	bufferPutKeyValue(buffer, TX_BLOCK_METADATA, container.TransactionsBlock.Metadata.Raw())

	for _, tx := range container.TransactionsBlock.SignedTransactions {
		bufferPutKeyValue(buffer, TX_BLOCK_SIGNED_TRANSACTION, tx.Raw())
	}

	bufferPutKeyValue(buffer, RS_BLOCK_HEADER, container.ResultsBlock.Header.Raw())
	bufferPutKeyValue(buffer, RS_BLOCK_PROOF, container.ResultsBlock.BlockProof.Raw())

	for _, receipt := range container.ResultsBlock.TransactionReceipts {
		bufferPutKeyValue(buffer, RS_BLOCK_TRANSACTION_RECEIPTS, receipt.Raw())
	}

	for _, stateDiff := range container.ResultsBlock.ContractStateDiffs {
		bufferPutKeyValue(buffer, RS_BLOCK_CONTRACT_STATE_DIFFS, stateDiff.Raw())
	}

	return buffer.Bytes()
}

func byteArrayAsBlock(data []byte) *protocol.BlockPairContainer {
	var txBlockHeaderRaw, txBlockProofRaw, txBlockMetadataRaw,
	rsBlockHeaderRaw, rsBlockProofRaw []byte

	var txSignedTransactionsRaw, rsStateDiffs, rsTransactionReceipts [][]byte

	iterateOverKeyValueBuffer(bytes.NewBuffer(data), func(key string, value []byte) {
		switch key {
		case TX_BLOCK_HEADER:
			txBlockHeaderRaw = value
		case TX_BLOCK_METADATA:
			txBlockMetadataRaw = value
		case TX_BLOCK_PROOF:
			txBlockProofRaw = value
		case TX_BLOCK_SIGNED_TRANSACTION:
			txSignedTransactionsRaw = append(txSignedTransactionsRaw, value)
		case RS_BLOCK_HEADER:
			rsBlockHeaderRaw = value
		case RS_BLOCK_PROOF:
			rsBlockProofRaw = value
		case RS_BLOCK_TRANSACTION_RECEIPTS:
			rsTransactionReceipts = append(rsTransactionReceipts, value)
		case RS_BLOCK_CONTRACT_STATE_DIFFS:
			rsStateDiffs = append(rsStateDiffs, value)
		}
	})

	return &protocol.BlockPairContainer{
		TransactionsBlock: constructTxBlockFromStorage(txBlockHeaderRaw, txBlockProofRaw, txBlockMetadataRaw, txSignedTransactionsRaw),
		ResultsBlock: constructResultsBlockFromStorage(rsBlockHeaderRaw, rsBlockProofRaw, rsStateDiffs, rsTransactionReceipts),
	}
}