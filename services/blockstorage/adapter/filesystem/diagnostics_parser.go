package filesystem

import (
	"bufio"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"io"
	"os"
)

type DiagnosticsParser struct {
	f                *os.File
	r                *bufio.Reader
	Header           blocksFileHeader
	firstBlockOffset int64
	FileInfo         os.FileInfo
}

func NewDiagnosticsParser(filepath string) (*DiagnosticsParser, error) {
	file, blocksOffset, header, err := openBlocksFileForDiagnostics(filepath)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return &DiagnosticsParser{
		f:                file,
		r:                bufio.NewReaderSize(file, 1024*1024),
		Header:           *header,
		firstBlockOffset: blocksOffset,
		FileInfo:         info,
	}, nil
}

func openBlocksFileForDiagnostics(filename string) (*os.File, int64, *blocksFileHeader, error) {

	file, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil {
		return nil, 0, nil, errors.Wrapf(err, "failed to open blocks file for reading %s", filename)
	}

	header := newBlocksFileHeader(0, 0)
	err = header.read(file)
	if err != nil {
		return nil, 0, nil, errors.Wrapf(err, "error reading blocks file Header")
	}

	offset, err := file.Seek(0, io.SeekCurrent) // read current offset
	if err != nil {
		return nil, 0, nil, errors.Wrapf(err, "error reading blocks file Header")
	}

	return file, offset, header, nil
}

func (dp *DiagnosticsParser) Close() {
	defer func() {
		recover()
	}()

	dp.f.Close()
}

func (dp *DiagnosticsParser) ScanFile(maxBlockSizeBytes uint32, f func(size int, offset int64, block *protocol.BlockPairContainer)) error {
	codec := newCodec(maxBlockSizeBytes)

	offset := dp.firstBlockOffset
	for {
		aBlock, blockSize, err := codec.decode(dp.r)
		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}
		f(blockSize, offset, aBlock)
		offset = offset + int64(blockSize)
	}
}
