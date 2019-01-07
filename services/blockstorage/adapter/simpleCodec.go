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

const codecVersion = 0

type simpleCodec struct {
	maxBlockSize int
}

func newSimpleCodec(maxBlockSize uint32) *simpleCodec {
	return &simpleCodec{
		maxBlockSize: int(maxBlockSize),
	}
}

type blockHeader struct {
	Version      uint32
	FixedSize    uint32
	ReceiptsSize uint32
	DiffsSize    uint32
	TxsSize      uint32
}

func diskChunkSize(bytes []byte) uint32 {
	return uint32(unsafe.Sizeof(uint32(0))) + uint32(len(bytes))
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

func (h *blockHeader) totalSize() uint32 {
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
	err := binary.Write(writer, binary.LittleEndian, uint32(len(message.Raw())))
	if err != nil {
		return err
	}
	err = binary.Write(writer, binary.LittleEndian, message.Raw())
	if err != nil {
		return err
	}
	return nil
}

func (c *simpleCodec) encode(block *protocol.BlockPairContainer, w io.Writer) error {
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

	if sw.bytesWritten > c.maxBlockSize {
		return fmt.Errorf("block size exceeds max limit")
	}

	// checksum
	err = binary.Write(w, binary.LittleEndian, checkSum.Sum32())
	if err != nil {
		return err
	}

	return nil
}

func (c *simpleCodec) decode(r io.Reader) (*protocol.BlockPairContainer, int, error) {
	checkSum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	tr := io.TeeReader(r, checkSum)

	serializationHeader := &blockHeader{}
	err := serializationHeader.read(tr)
	if err != nil {
		return nil, 0, err
	}

	headerSize := int(unsafe.Sizeof(*serializationHeader))
	budget := newReadingBudget(
		int(serializationHeader.totalSize())+headerSize,
		headerSize)

	if budget.limit > c.maxBlockSize {
		return nil, budget.bytesRead, fmt.Errorf("block size exceeds max limit")
	}

	if serializationHeader.Version != codecVersion {
		return nil, budget.bytesRead, fmt.Errorf("encountered unsupported codec version %d", serializationHeader.Version)
	}

	tbHeaderChunk, err := readChunk(tr, budget)
	if err != nil {
		return nil, budget.bytesRead, err
	}
	tbHeader := protocol.TransactionsBlockHeaderReader(tbHeaderChunk)

	tbMetadataChunk, err := readChunk(tr, budget)
	if err != nil {
		return nil, budget.bytesRead, err
	}
	tbMetadata := protocol.TransactionsBlockMetadataReader(tbMetadataChunk)

	tbBlockProofChunk, err := readChunk(tr, budget)
	if err != nil {
		return nil, budget.bytesRead, err
	}
	tbBlockProof := protocol.TransactionsBlockProofReader(tbBlockProofChunk)

	rbHeaderChunk, err := readChunk(tr, budget)
	if err != nil {
		return nil, budget.bytesRead, err
	}
	rbHeader := protocol.ResultsBlockHeaderReader(rbHeaderChunk)

	rbBlockProofChunk, err := readChunk(tr, budget)
	if err != nil {
		return nil, budget.bytesRead, err
	}
	rbBlockProof := protocol.ResultsBlockProofReader(rbBlockProofChunk)

	receipts := make([]*protocol.TransactionReceipt, 0, rbHeader.NumTransactionReceipts())
	for i := 0; i < cap(receipts); i++ {
		chunk, err := readChunk(tr, budget)
		if err != nil {
			return nil, budget.bytesRead, err
		}
		receipts = append(receipts, protocol.TransactionReceiptReader(chunk))
	}

	stateDiffs := make([]*protocol.ContractStateDiff, 0, rbHeader.NumContractStateDiffs())
	for i := 0; i < cap(stateDiffs); i++ {
		chunk, err := readChunk(tr, budget)
		if err != nil {
			return nil, budget.bytesRead, err
		}
		stateDiffs = append(stateDiffs, protocol.ContractStateDiffReader(chunk))
	}

	txs := make([]*protocol.SignedTransaction, 0, tbHeader.NumSignedTransactions())
	for i := 0; i < cap(txs); i++ {
		chunk, err := readChunk(tr, budget)
		if err != nil {
			return nil, budget.bytesRead, err
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

	if budget.bytesRead != budget.limit {
		return nil, budget.bytesRead, fmt.Errorf("block size mismatch. expected: %v read: %v", budget.limit, budget.bytesRead)
	}

	// checksum
	var readCheckSum uint32
	err = binary.Read(r, binary.LittleEndian, &readCheckSum)
	if err != nil {
		return nil, budget.bytesRead, err
	}

	if readCheckSum != checkSum.Sum32() {
		return nil, budget.bytesRead + int(unsafe.Sizeof(readCheckSum)), fmt.Errorf("block checksum mismatch. computed: %v recorded: %v", checkSum.Sum32(), readCheckSum)
	}

	return blockPair, budget.bytesRead + int(unsafe.Sizeof(readCheckSum)), nil
}

type checksumWriter struct {
	w, checksum  io.Writer
	bytesWritten int
}

func newChecksumWriter(w, checksum io.Writer) *checksumWriter {
	return &checksumWriter{w: w, checksum: checksum}
}

func (cw *checksumWriter) Write(p []byte) (int, error) {
	_, err := cw.checksum.Write(p)
	if err != nil {
		return 0, errors.Wrap(err, "failed adding to checksum")
	}
	n, err := cw.w.Write(p)
	cw.bytesWritten += n
	return n, err
}

type readingBudget struct {
	limit     int
	bytesRead int
}

func newReadingBudget(limit int, bytesRead int) *readingBudget {
	return &readingBudget{
		limit:     limit,
		bytesRead: bytesRead,
	}
}

func readChunk(reader io.Reader, budget *readingBudget) ([]byte, error) {
	var chunkSize uint32
	err := binary.Read(reader, binary.LittleEndian, &chunkSize)
	if err != nil {
		return nil, err
	}
	budget.bytesRead += int(unsafe.Sizeof(chunkSize))

	// check budget
	if budget.limit < budget.bytesRead+int(chunkSize) {
		return nil, fmt.Errorf("invalid block. size exceeds limit (%d)", budget.limit)
	}

	chunk := make([]byte, chunkSize)
	n, err := reader.Read(chunk)
	if err != nil {
		return nil, err
	}
	if n != len(chunk) {
		return nil, fmt.Errorf("read %d bytes in block chuck while expecting %d", n, len(chunk))
	}

	budget.bytesRead += n
	return chunk, nil
}
