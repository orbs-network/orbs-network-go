// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package filesystem

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

const orbsFormatMagic = uint32(0x5342524f) // "ORBS"
const orbsFormatVersion = 0
const blockMagic = uint32(0x6b4f4c42) // "BLOk"
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
	Magic       uint32
	FileVersion uint32
	NetworkId   uint32
	ChainId     uint32
}

type blockHeader struct {
	Magic        uint32
	Version      uint32
	FixedSize    uint32
	ReceiptsSize uint32
	DiffsSize    uint32
	TxsSize      uint32
}

func diskChunkSize(bytes []byte) uint32 {
	return uint32(chunkLengthSize) + uint32(len(bytes))
}

func newBlockHeader() *blockHeader {
	return &blockHeader{
		Magic:   blockMagic,
		Version: blockVersion,
	}
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

	if bh.Magic != blockMagic {
		return fmt.Errorf("invalid block magic number %v", bh.Magic)
	}

	if bh.Version != blockVersion {
		return fmt.Errorf("invalid block version %d", bh.Version)
	}

	return nil
}

func newBlocksFileHeader(networkId, vchainId uint32) *blocksFileHeader {
	return &blocksFileHeader{
		Magic:       orbsFormatMagic,
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

	if bfh.Magic != orbsFormatMagic {
		return fmt.Errorf("invalid magic number %v", bfh.Magic)
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

	// calc header
	blockHeader := newBlockHeader()
	blockHeader.addFixed(tb.Header)
	blockHeader.addFixed(tb.Metadata)
	blockHeader.addFixed(tb.BlockProof)
	blockHeader.addFixed(rb.Header)
	blockHeader.addFixed(rb.BlockProof)

	for _, receipt := range rb.TransactionReceipts {
		blockHeader.addReceipt(receipt)
	}
	for _, diff := range rb.ContractStateDiffs {
		blockHeader.addDiff(diff)
	}
	for _, tx := range tb.SignedTransactions {
		blockHeader.addTx(tx)
	}

	fullBlockChecksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	fullBlockWriter := newChecksumWriter(w, fullBlockChecksum)
	err := blockHeader.write(fullBlockWriter)
	if err != nil {
		return 0, err
	}

	err = c.writeFixedBlockSectionWithChecksum(fullBlockWriter, block)
	if err != nil {
		return 0, err
	}

	err = c.writeDynamicBlockSectionWithChecksum(fullBlockWriter, transactionReceiptsToMessages(rb.TransactionReceipts))
	if err != nil {
		return 0, err
	}

	err = c.writeDynamicBlockSectionWithChecksum(fullBlockWriter, diffsToMessages(rb.ContractStateDiffs))
	if err != nil {
		return 0, err
	}

	err = c.writeDynamicBlockSectionWithChecksum(fullBlockWriter, transactionsToMessages(tb.SignedTransactions))
	if err != nil {
		return 0, err
	}

	if fullBlockWriter.bytesWritten > c.maxBlockSize { // check if we exceeded budget
		return 0, fmt.Errorf("block size exceeds max limit. wrote %d bytes", fullBlockWriter.bytesWritten)
	}

	err = binary.Write(w, binary.LittleEndian, fullBlockChecksum.Sum32()) // full block checksum
	if err != nil {
		return 0, err
	}

	return blockHeaderSize + blockHeader.totalSize() + checksumSize*5, nil
}

func (c *codec) writeFixedBlockSectionWithChecksum(w io.Writer, block *protocol.BlockPairContainer) error {
	tb := block.TransactionsBlock
	rb := block.ResultsBlock

	fixedSectionChecksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	fixedSectionWriter := newChecksumWriter(w, fixedSectionChecksum)

	err := writeMessage(fixedSectionWriter, tb.Header)
	if err != nil {
		return err
	}
	err = writeMessage(fixedSectionWriter, tb.Metadata)
	if err != nil {
		return err
	}
	err = writeMessage(fixedSectionWriter, tb.BlockProof)
	if err != nil {
		return err
	}
	err = writeMessage(fixedSectionWriter, rb.Header)
	if err != nil {
		return err
	}
	err = writeMessage(fixedSectionWriter, rb.BlockProof)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, fixedSectionChecksum.Sum32()) // fixed section checksum
	if err != nil {
		return err
	}
	return nil
}

func (c *codec) writeDynamicBlockSectionWithChecksum(w io.Writer, messages []membuffers.Message) error {
	sectionChecksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	sectionWriter := newChecksumWriter(w, sectionChecksum)

	for _, message := range messages {
		err := writeMessage(sectionWriter, message)
		if err != nil {
			return err
		}
	}

	sum32 := sectionChecksum.Sum32()
	err := binary.Write(w, binary.LittleEndian, sum32) // section checksum
	if err != nil {
		return err
	}
	return nil
}

// TODO V1 see https://tree.taiga.io/project/orbs-network/us/681
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

	fixed, _, err := c.readFixedSection(tr, budget)
	if err != nil {
		return nil, budget.bytesRead, err
	}

	receipts, _, err := c.readReceiptsSection(tr, budget, fixed.resultsBlockHeader.NumTransactionReceipts())
	if err != nil {
		return nil, budget.bytesRead, err
	}

	stateDiffs, _, err := c.readStateDiffsSection(tr, budget, fixed.resultsBlockHeader.NumContractStateDiffs())
	if err != nil {
		return nil, budget.bytesRead, err
	}

	txs, _, err := c.readTransactionsSection(tr, budget, fixed.transactionsBlockHeader.NumSignedTransactions())
	if err != nil {
		return nil, budget.bytesRead, err
	}

	if budget.bytesRead != budget.limit {
		return nil, budget.bytesRead, fmt.Errorf("block size mismatch. expected: %v read: %v", budget.limit, budget.bytesRead)
	}

	var checksum uint32
	err = binary.Read(r, binary.LittleEndian, &checksum)
	if err != nil {
		return nil, budget.bytesRead, err
	}

	if checksum != checkSum.Sum32() {
		return nil, budget.bytesRead + checksumSize, fmt.Errorf("block checksum mismatch. computed: %v recorded: %v", checkSum.Sum32(), checksum)
	}

	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header:             fixed.transactionsBlockHeader,
			Metadata:           fixed.transactionsBlockMetadata,
			SignedTransactions: txs,
			BlockProof:         fixed.transactionsBlockProof,
		},
		ResultsBlock: &protocol.ResultsBlockContainer{
			Header:              fixed.resultsBlockHeader,
			TransactionReceipts: receipts,
			ContractStateDiffs:  stateDiffs,
			BlockProof:          fixed.resultsBlockProof,
		},
	}

	return blockPair, budget.bytesRead + checksumSize*5, nil
}

type fixedSizeBlockSection struct {
	transactionsBlockHeader   *protocol.TransactionsBlockHeader
	transactionsBlockMetadata *protocol.TransactionsBlockMetadata
	transactionsBlockProof    *protocol.TransactionsBlockProof
	resultsBlockHeader        *protocol.ResultsBlockHeader
	resultsBlockProof         *protocol.ResultsBlockProof
}

func (c *codec) readFixedSection(r io.Reader, budget *readingBudget) (*fixedSizeBlockSection, uint32, error) {
	fixed := &fixedSizeBlockSection{}

	tbHeaderChunk, err := readChunk(r, budget)
	if err != nil {
		return nil, 0, err
	}
	fixed.transactionsBlockHeader = protocol.TransactionsBlockHeaderReader(tbHeaderChunk)

	tbMetadataChunk, err := readChunk(r, budget)
	if err != nil {
		return nil, 0, err
	}
	fixed.transactionsBlockMetadata = protocol.TransactionsBlockMetadataReader(tbMetadataChunk)

	tbBlockProofChunk, err := readChunk(r, budget)
	if err != nil {
		return nil, 0, err
	}
	fixed.transactionsBlockProof = protocol.TransactionsBlockProofReader(tbBlockProofChunk)

	rbHeaderChunk, err := readChunk(r, budget)
	if err != nil {
		return nil, 0, err
	}
	fixed.resultsBlockHeader = protocol.ResultsBlockHeaderReader(rbHeaderChunk)

	rbBlockProofChunk, err := readChunk(r, budget)
	if err != nil {
		return nil, 0, err
	}
	fixed.resultsBlockProof = protocol.ResultsBlockProofReader(rbBlockProofChunk)

	var checksum uint32
	err = binary.Read(r, binary.LittleEndian, &checksum)
	if err != nil {
		return nil, 0, err
	}

	return fixed, checksum, nil
}

func (c *codec) readReceiptsSection(tr io.Reader, budget *readingBudget, count uint32) ([]*protocol.TransactionReceipt, uint32, error) {
	chunks, checksum, err := c.readDynamicBlockSection(tr, budget, count)
	if err != nil {
		return nil, 0, err
	}
	receipts := make([]*protocol.TransactionReceipt, 0, len(chunks))
	for _, chunk := range chunks {
		receipts = append(receipts, protocol.TransactionReceiptReader(chunk))
	}

	return receipts, checksum, err
}

func (c *codec) readStateDiffsSection(tr io.Reader, budget *readingBudget, count uint32) ([]*protocol.ContractStateDiff, uint32, error) {
	chunks, checksum, err := c.readDynamicBlockSection(tr, budget, count)
	if err != nil {
		return nil, 0, err
	}
	receipts := make([]*protocol.ContractStateDiff, 0, len(chunks))
	for _, chunk := range chunks {
		receipts = append(receipts, protocol.ContractStateDiffReader(chunk))
	}

	return receipts, checksum, err
}

func (c *codec) readTransactionsSection(tr io.Reader, budget *readingBudget, count uint32) ([]*protocol.SignedTransaction, uint32, error) {
	chunks, checksum, err := c.readDynamicBlockSection(tr, budget, count)
	if err != nil {
		return nil, 0, err
	}
	receipts := make([]*protocol.SignedTransaction, 0, len(chunks))
	for _, chunk := range chunks {
		receipts = append(receipts, protocol.SignedTransactionReader(chunk))
	}

	return receipts, checksum, err
}

func (c *codec) readDynamicBlockSection(tr io.Reader, budget *readingBudget, count uint32) ([][]byte, uint32, error) {
	if uint32(budget.limit) < count {
		return nil, 0, fmt.Errorf("section element count is invalid. attempting to read %d elements in block section while size budget is only %d", count, budget.limit)
	}

	chunks := make([][]byte, 0, count)
	for i := 0; i < cap(chunks); i++ {
		chunk, err := readChunk(tr, budget)
		if err != nil {
			return nil, 0, err
		}
		chunks = append(chunks, chunk)
	}

	var checksum uint32
	err := binary.Read(tr, binary.LittleEndian, &checksum)
	if err != nil {
		return nil, 0, err
	}
	return chunks, checksum, err
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

// TODO V1 will be deleted when: https://tree.taiga.io/project/orbs-network/us/681
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

func transactionReceiptsToMessages(receipts []*protocol.TransactionReceipt) (messages []membuffers.Message) {
	messages = make([]membuffers.Message, 0, len(receipts))
	for _, receipt := range receipts {
		messages = append(messages, receipt)
	}
	return messages
}

func diffsToMessages(diffs []*protocol.ContractStateDiff) (messages []membuffers.Message) {
	messages = make([]membuffers.Message, 0, len(diffs))
	for _, diff := range diffs {
		messages = append(messages, diff)
	}
	return messages
}

func transactionsToMessages(txs []*protocol.SignedTransaction) (messages []membuffers.Message) {
	messages = make([]membuffers.Message, 0, len(txs))
	for _, tx := range txs {
		messages = append(messages, tx)
	}
	return messages
}
