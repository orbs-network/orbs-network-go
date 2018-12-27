package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"io"
	"os"
	"sync"
)

type writingTip struct {
	sync.Mutex
	file       *os.File
	currentPos int64
	logger     log.BasicLogger
}

func newWritingTip(ctx context.Context, dir, filename string, logger log.BasicLogger) (*writingTip, error) {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify data directory exists")
	}
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open blocks file for writing")
	}
	result := &writingTip{
		file:   file,
		logger: logger,
	}

	go func() {
		<-ctx.Done()
		result.close()
	}()

	return result, nil
}

func (wt *writingTip) writeBlockAtOffset(pos int64, blockPair *protocol.BlockPairContainer) (int64, error) {
	if pos != wt.currentPos {
		currentOffset, err := wt.file.Seek(pos, io.SeekStart)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to seek writing tip to pos %d", pos)
		}
		if pos != currentOffset {
			return 0, errors.Wrapf(err, "failed to seek in blocks file to position %v", pos)
		}
	}

	err := encode(blockPair, wt.file)
	if err != nil {
		return 0, errors.Wrap(err, "failed to write block")
	}

	err = wt.file.Sync()
	if err != nil {
		return 0, errors.Wrap(err, "failed to flush blocks file to disk")
	}
	// find our current offset
	newPos, err := wt.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, errors.Wrap(err, "failed to update block height index")
	}
	wt.currentPos = newPos // assign only after checking err
	return newPos, nil
}

func (wt *writingTip) close() {
	wt.Lock()
	defer wt.Unlock()
	err := wt.file.Close()
	if err != nil {
		wt.logger.Error("failed to close blocks file", log.String("filename", wt.file.Name()))
		return
	}
	wt.logger.Info("closed blocks file", log.String("filename", wt.file.Name()))
}
