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

const blockHeaderSize = int(unsafe.Sizeof(blockHeader{}))
const checksumSize = int(unsafe.Sizeof(uint32(0)))
const chunkLengthSize = int(unsafe.Sizeof(uint32(0)))

const orbsFormatMagic = uint32(0x5342524f)
const orbsFormatVersion = 0
const blockVersion = 0

type codec struct {
	maxBlockSize int
}

func newCodec(maxBlockSize uint32) *codec {
	return &codec{
		maxBlockSize: int(maxBlockSize),
	}
}

type blocksFileHeader struct {
	ORBS        uint32
	FileVersion uint32
	NetworkId   uint32
	ChainId     uint32
}

type blockHeader struct {
	Version      uint32
	FixedSize    uint32
	ReceiptsSize uint32
	DiffsSize    uint32
	TxsSize      uint32
}

func diskChunkSize(bytes []byte) uint32 {
	return uint32(chunkLengthSize) + uint32(len(bytes))
}

func (bh *blockHeader) addFixed(m membuffers.Message) {
	bh.FixedSize += diskChunkSize(m.Raw())
}

func (bh *blockHeader) addReceipt(receipt *protocol.TransactionReceipt) {
	bh.ReceiptsSize += diskChunkSize(receipt.Raw())
}

func (bh *blockHeader) addDiff(diff *protocol.ContractStateDiff) {
	bh.DiffsSize += diskChunkSize(diff.Raw())
}

func (bh *blockHeader) addTx(tx *protocol.SignedTransaction) {
	bh.TxsSize += diskChunkSize(tx.Raw())
}

func (bh *blockHeader) totalSize() int {
	return int(bh.FixedSize + bh.DiffsSize + bh.ReceiptsSize + bh.TxsSize)
}

func (bh *blockHeader) write(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, bh)
	if err != nil {
		return err
	}
	return nil
}

func (bh *blockHeader) read(r io.Reader) error {
	err := binary.Read(r, binary.LittleEndian, bh)
	if err != nil {
		return err
	}
	return nil
}

func newBlocksFileHeader(networkId, vchainId uint32) *blocksFileHeader {
	return &blocksFileHeader{
		ORBS:        orbsFormatMagic,
		FileVersion: orbsFormatVersion,
		NetworkId:   networkId,
		ChainId:     vchainId,
	}
}

func (bfh *blocksFileHeader) read(r io.Reader) error {
	checkSum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	tr := io.TeeReader(r, checkSum)
	err := binary.Read(tr, binary.LittleEndian, bfh)
	if err != nil {
		return err
	}

	// checksum
	var sum32 uint32
	err = binary.Read(r, binary.LittleEndian, &sum32)
	if err != nil {
		return errors.Wrapf(err, "failed reading header checksum")
	}

	if sum32 != checkSum.Sum32() {
		return fmt.Errorf("invalid header, bad checksum")
	}

	if bfh.ORBS != orbsFormatMagic {
		return fmt.Errorf("invalid magic number %v", orbsFormatMagic)
	}
	if bfh.FileVersion != orbsFormatVersion {
		return fmt.Errorf("invalid version %d", bfh.FileVersion)
	}
	return nil
}

func (bfh *blocksFileHeader) write(w io.Writer) error {
	checkSum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	sw := newChecksumWriter(w, checkSum)

	err := binary.Write(sw, binary.LittleEndian, bfh)
	if err != nil {
		return err
	}

	sum32 := checkSum.Sum32()
	err = binary.Write(w, binary.LittleEndian, sum32)
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

func (c *codec) encode(block *protocol.BlockPairContainer, w io.Writer) (int, error) {
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
		return 0, err
	}

	// write buffers
	err = writeMessage(sw, tb.Header)
	if err != nil {
		return 0, err
	}
	err = writeMessage(sw, tb.Metadata)
	if err != nil {
		return 0, err
	}
	err = writeMessage(sw, tb.BlockProof)
	if err != nil {
		return 0, err
	}
	err = writeMessage(sw, rb.Header)
	if err != nil {
		return 0, err
	}
	err = writeMessage(sw, rb.BlockProof)
	if err != nil {
		return 0, err
	}

	for _, receipt := range rb.TransactionReceipts {
		err = writeMessage(sw, receipt)
		if err != nil {
			return 0, err
		}

	}
	for _, diff := range rb.ContractStateDiffs {
		err = writeMessage(sw, diff)
		if err != nil {
			return 0, err
		}
	}
	for _, tx := range tb.SignedTransactions {
		err = writeMessage(sw, tx)
		if err != nil {
			return 0, err
		}
	}

	if sw.bytesWritten > c.maxBlockSize {
		return 0, fmt.Errorf("block size exceeds max limit. wrote %d bytes", sw.bytesWritten)
	}

	// checksum
	err = binary.Write(w, binary.LittleEndian, checkSum.Sum32())
	if err != nil {
		return 0, err
	}

	return blockHeaderSize + serializationHeader.totalSize() + checksumSize, nil
}

func (c *codec) decode(r io.Reader) (*protocol.BlockPairContainer, int, error) {
	checkSum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	tr := io.TeeReader(r, checkSum)

	serializationHeader := &blockHeader{}
	err := serializationHeader.read(tr)
	if err != nil {
		return nil, 0, err
	}

	budget := newReadingBudget(
		int(serializationHeader.totalSize())+blockHeaderSize,
		blockHeaderSize)

	if budget.limit > c.maxBlockSize {
		return nil, budget.bytesRead, fmt.Errorf("block size exceeds max limit. block header %#v", serializationHeader)
	}

	if serializationHeader.Version != blockVersion {
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
		return nil, budget.bytesRead + checksumSize, fmt.Errorf("block checksum mismatch. computed: %v recorded: %v", checkSum.Sum32(), readCheckSum)
	}

	return blockPair, budget.bytesRead + checksumSize, nil
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
	var chunkLength uint32
	err := binary.Read(reader, binary.LittleEndian, &chunkLength)
	if err != nil {
		return nil, err
	}
	budget.bytesRead += chunkLengthSize

	// check budget
	if budget.limit < budget.bytesRead+int(chunkLength) {
		return nil, fmt.Errorf("invalid block. size exceeds limit (%d)", budget.limit)
	}

	chunk := make([]byte, chunkLength)
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
