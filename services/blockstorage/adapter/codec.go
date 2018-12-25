package adapter

import (
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"io"
	"unsafe"
)

// TODO V1 write test for violation of: blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.ResultsBlock == nil
// TODO V1 write test for violation of: other nil values

// TODO V1 write codec version in header or maybe file header?
// TODO V1 remove unneeded fields from header.
type chunkSize uint32

const chunkSizeBytes = 4

type blockHeader struct {
	FixedSize    chunkSize
	ReceiptsSize chunkSize
	DiffsSize    chunkSize
	TxsSize      chunkSize
}

func diskChunkSize(bytes []byte) chunkSize {
	return chunkSizeBytes + chunkSize(len(bytes))
}

func (h *blockHeader) addFixed(m membuffers.Message) {
	h.FixedSize += diskChunkSize(m.Raw())
}

func (h *blockHeader) addReceipt(receipt *protocol.TransactionReceipt) {
	h.ReceiptsSize += diskChunkSize(receipt.Raw())
}

func (h *blockHeader) addDiff(diff *protocol.ContractStateDiff) {
	h.DiffsSize += diskChunkSize(diff.Raw())
}

func (h *blockHeader) addTx(tx *protocol.SignedTransaction) {
	h.TxsSize += diskChunkSize(tx.Raw())
}

func (h *blockHeader) totalSize() chunkSize {
	return h.FixedSize + h.DiffsSize + h.ReceiptsSize + h.TxsSize
}

func (h *blockHeader) write(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, h)
	if err != nil {
		return err
	}
	return nil
}

func (h *blockHeader) read(r io.Reader) error {
	err := binary.Read(r, binary.LittleEndian, h)
	if err != nil {
		return err
	}
	return nil
}

func writeMessage(writer io.Writer, message membuffers.Message) error {
	err := binary.Write(writer, binary.LittleEndian, chunkSize(len(message.Raw())))
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.LittleEndian, message.Raw())
	if err != nil {
		return err
	}
	return nil
}

func readChunk(reader io.Reader, byteCounter *int) ([]byte, error) {
	var chunkSize chunkSize
	err := binary.Read(reader, binary.LittleEndian, &chunkSize)
	if err != nil {
		return nil, err
	}

	chunk := make([]byte, chunkSize)
	n, err := reader.Read(chunk)
	if err != nil {
		return nil, err
	}
	if n != len(chunk) {
		return nil, fmt.Errorf("read %d bytes in block chuck while expecting %d", n, len(chunk))
	}

	*byteCounter += n + chunkSizeBytes
	return chunk, nil
}

func encode(block *protocol.BlockPairContainer, w io.Writer) error {
	tb := block.TransactionsBlock
	rb := block.ResultsBlock

	// write header
	serializationHeader := &blockHeader{}
	serializationHeader.addFixed(tb.Header)
	serializationHeader.addFixed(tb.Metadata)
	serializationHeader.addFixed(tb.BlockProof)
	serializationHeader.addFixed(rb.Header)
	serializationHeader.addFixed(rb.BlockProof)
	serializationHeader.addFixed(rb.TransactionsBloomFilter)
	for _, receipt := range rb.TransactionReceipts {
		serializationHeader.addReceipt(receipt)
	}
	for _, diff := range rb.ContractStateDiffs {
		serializationHeader.addDiff(diff)
	}
	for _, tx := range tb.SignedTransactions {
		serializationHeader.addTx(tx)
	}

	err := serializationHeader.write(w)

	if err != nil {
		return err
	}

	// write buffers
	err = writeMessage(w, tb.Header)
	if err != nil {
		return err
	}
	err = writeMessage(w, tb.Metadata)
	if err != nil {
		return err
	}
	err = writeMessage(w, tb.BlockProof)
	if err != nil {
		return err
	}
	err = writeMessage(w, rb.Header)
	if err != nil {
		return err
	}
	err = writeMessage(w, rb.BlockProof)
	if err != nil {
		return err
	}
	err = writeMessage(w, rb.TransactionsBloomFilter)
	if err != nil {
		return err
	}

	for _, receipt := range rb.TransactionReceipts {
		err = writeMessage(w, receipt)
		if err != nil {
			return err
		}

	}
	for _, diff := range rb.ContractStateDiffs {
		err = writeMessage(w, diff)
		if err != nil {
			return err
		}
	}
	for _, tx := range tb.SignedTransactions {
		err = writeMessage(w, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func decode(r io.Reader) (*protocol.BlockPairContainer, int, error) {
	serializationHeader := &blockHeader{}
	err := serializationHeader.read(r)
	if err != nil {
		return nil, 0, err
	}

	byteCounter := int(unsafe.Sizeof(*serializationHeader))
	tbHeaderChunk, err := readChunk(r, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	tbHeader := protocol.TransactionsBlockHeaderReader(tbHeaderChunk)

	tbMetadataChunk, err := readChunk(r, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	tbMetadata := protocol.TransactionsBlockMetadataReader(tbMetadataChunk)

	tbBlockProofChunk, err := readChunk(r, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	tbBlockProof := protocol.TransactionsBlockProofReader(tbBlockProofChunk)

	rbHeaderChunk, err := readChunk(r, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	rbHeader := protocol.ResultsBlockHeaderReader(rbHeaderChunk)

	rbBlockProofChunk, err := readChunk(r, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	rbBlockProof := protocol.ResultsBlockProofReader(rbBlockProofChunk)

	rbBloomChunk, err := readChunk(r, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	rbBloomFilter := protocol.TransactionsBloomFilterReader(rbBloomChunk)

	// TODO V1 add validations : - 1) IsValid() on each membuff 2) check that num of bytes read match header
	receipts := make([]*protocol.TransactionReceipt, 0, rbHeader.NumTransactionReceipts())
	for i := 0; i < cap(receipts); i++ {
		chunk, err := readChunk(r, &byteCounter)
		if err != nil {
			return nil, byteCounter, err
		}
		receipts = append(receipts, protocol.TransactionReceiptReader(chunk))
	}

	stateDiffs := make([]*protocol.ContractStateDiff, 0, rbHeader.NumContractStateDiffs())
	for i := 0; i < cap(stateDiffs); i++ {
		chunk, err := readChunk(r, &byteCounter)
		if err != nil {
			return nil, byteCounter, err
		}
		stateDiffs = append(stateDiffs, protocol.ContractStateDiffReader(chunk))
	}

	txs := make([]*protocol.SignedTransaction, 0, tbHeader.NumSignedTransactions())
	for i := 0; i < cap(txs); i++ {
		chunk, err := readChunk(r, &byteCounter)
		if err != nil {
			return nil, byteCounter, err
		}
		txs = append(txs, protocol.SignedTransactionReader(chunk))
	}

	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header:             tbHeader,
			Metadata:           tbMetadata,
			SignedTransactions: txs,
			BlockProof:         tbBlockProof,
		},
		ResultsBlock: &protocol.ResultsBlockContainer{
			Header:                  rbHeader,
			TransactionsBloomFilter: rbBloomFilter,
			TransactionReceipts:     receipts,
			ContractStateDiffs:      stateDiffs,
			BlockProof:              rbBlockProof,
		},
	}

	return blockPair, byteCounter, nil
}
