package adapter

import (
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"hash/crc32"
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
	for _, receipt := range rb.TransactionReceipts {
		serializationHeader.addReceipt(receipt)
	}
	for _, diff := range rb.ContractStateDiffs {
		serializationHeader.addDiff(diff)
	}
	for _, tx := range tb.SignedTransactions {
		serializationHeader.addTx(tx)
	}

	checkSum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	sw := newChecksumWriter(w, checkSum)
	err := serializationHeader.write(sw)
	if err != nil {
		return err
	}

	// write buffers
	err = writeMessage(sw, tb.Header)
	if err != nil {
		return err
	}
	err = writeMessage(sw, tb.Metadata)
	if err != nil {
		return err
	}
	err = writeMessage(sw, tb.BlockProof)
	if err != nil {
		return err
	}
	err = writeMessage(sw, rb.Header)
	if err != nil {
		return err
	}
	err = writeMessage(sw, rb.BlockProof)
	if err != nil {
		return err
	}

	for _, receipt := range rb.TransactionReceipts {
		err = writeMessage(sw, receipt)
		if err != nil {
			return err
		}

	}
	for _, diff := range rb.ContractStateDiffs {
		err = writeMessage(sw, diff)
		if err != nil {
			return err
		}
	}
	for _, tx := range tb.SignedTransactions {
		err = writeMessage(sw, tx)
		if err != nil {
			return err
		}
	}

	// checksum
	err = binary.Write(w, binary.LittleEndian, checkSum.Sum32())
	if err != nil {
		return err
	}

	return nil
}

func decode(r io.Reader) (*protocol.BlockPairContainer, int, error) {
	checkSum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	tr := io.TeeReader(r, checkSum)

	serializationHeader := &blockHeader{}
	err := serializationHeader.read(tr)
	if err != nil {
		return nil, 0, err
	}

	byteCounter := int(unsafe.Sizeof(*serializationHeader))
	tbHeaderChunk, err := readChunk(tr, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	tbHeader := protocol.TransactionsBlockHeaderReader(tbHeaderChunk)

	tbMetadataChunk, err := readChunk(tr, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	tbMetadata := protocol.TransactionsBlockMetadataReader(tbMetadataChunk)

	tbBlockProofChunk, err := readChunk(tr, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	tbBlockProof := protocol.TransactionsBlockProofReader(tbBlockProofChunk)

	rbHeaderChunk, err := readChunk(tr, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	rbHeader := protocol.ResultsBlockHeaderReader(rbHeaderChunk)

	rbBlockProofChunk, err := readChunk(tr, &byteCounter)
	if err != nil {
		return nil, byteCounter, err
	}
	rbBlockProof := protocol.ResultsBlockProofReader(rbBlockProofChunk)

	// TODO V1 add validations : - 1) IsValid() on each membuff 2) check that num of bytes read match header
	receipts := make([]*protocol.TransactionReceipt, 0, rbHeader.NumTransactionReceipts())
	for i := 0; i < cap(receipts); i++ {
		chunk, err := readChunk(tr, &byteCounter)
		if err != nil {
			return nil, byteCounter, err
		}
		receipts = append(receipts, protocol.TransactionReceiptReader(chunk))
	}

	stateDiffs := make([]*protocol.ContractStateDiff, 0, rbHeader.NumContractStateDiffs())
	for i := 0; i < cap(stateDiffs); i++ {
		chunk, err := readChunk(tr, &byteCounter)
		if err != nil {
			return nil, byteCounter, err
		}
		stateDiffs = append(stateDiffs, protocol.ContractStateDiffReader(chunk))
	}

	txs := make([]*protocol.SignedTransaction, 0, tbHeader.NumSignedTransactions())
	for i := 0; i < cap(txs); i++ {
		chunk, err := readChunk(tr, &byteCounter)
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
			Header:              rbHeader,
			TransactionReceipts: receipts,
			ContractStateDiffs:  stateDiffs,
			BlockProof:          rbBlockProof,
		},
	}

	var readCheckSum uint32
	err = binary.Read(r, binary.LittleEndian, &readCheckSum)
	if err != nil {
		return nil, byteCounter, err
	}
	byteCounter += int(unsafe.Sizeof(readCheckSum))

	if readCheckSum != checkSum.Sum32() {
		return nil, byteCounter, fmt.Errorf("block checksum mismatch. computed: %v recorded: %v", checkSum.Sum32(), readCheckSum)
	}
	return blockPair, byteCounter, nil
}

type checksumWriter struct {
	w, checksum io.Writer
}

func newChecksumWriter(w, checksum io.Writer) *checksumWriter {
	return &checksumWriter{w: w, checksum: checksum}
}

func (cw *checksumWriter) Write(p []byte) (int, error) {
	_, err := cw.checksum.Write(p)
	if err != nil {
		return 0, errors.Wrap(err, "failed adding to checksum")
	}
	return cw.w.Write(p)
}
