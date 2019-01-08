package adapter

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"io"
	"sync"
)

type writerSyncer interface {
	io.Writer
	Sync() error
}

type blockWriter struct {
	sync.Mutex
	ws     writerSyncer
	logger log.BasicLogger
	codec  blockCodec
}

func newBlockWriter(ws writerSyncer, logger log.BasicLogger, codec blockCodec) *blockWriter {
	return &blockWriter{
		ws:     ws,
		logger: logger,
		codec:  codec,
	}
}

func (bw *blockWriter) writeBlock(blockPair *protocol.BlockPairContainer) (int, error) {
	bytes, err := bw.codec.encode(blockPair, bw.ws)
	if err != nil {
		return 0, errors.Wrap(err, "failed to write block")
	}

	err = bw.ws.Sync()
	if err != nil {
		return 0, errors.Wrap(err, "failed to flush blocks to disk")
	}

	return bytes, nil
}
