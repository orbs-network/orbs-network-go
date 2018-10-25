package statestorage

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type revisionDiff struct {
	diff       adapter.ChainState
	merkleRoot primitives.MerkleSha256
	height     primitives.BlockHeight
	ts         primitives.TimestampNano
}

type rollingRevisions struct {
	prst               adapter.StatePersistence
	transientRevisions int
	revisions          []*revisionDiff
	currentHeight      primitives.BlockHeight
	currentTs          primitives.TimestampNano
	prstHight          primitives.BlockHeight
	prstRoot           primitives.MerkleSha256
	prstTs             primitives.TimestampNano
}

func newRollingRevisions(prst adapter.StatePersistence, transientRevisions int) *rollingRevisions {
	h, ts, r, err := prst.ReadMetadata()
	if err != nil {
		panic("could not load state metadata")
	}
	result := &rollingRevisions{
		prst:               prst,
		transientRevisions: transientRevisions,
		currentHeight:      h,
		currentTs:          ts,
		prstHight:          h,
		prstTs:             ts,
		prstRoot:           r,
	}

	return result
}

func (ls *rollingRevisions) getCurrentHeight() primitives.BlockHeight {
	return ls.currentHeight
}

func (ls *rollingRevisions) getCurrentTimestamp() primitives.TimestampNano {
	return ls.currentTs
}

func (ls *rollingRevisions) addRevision(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff adapter.ChainState) ([]primitives.MerkleSha256, error) {
	ls.revisions = append(ls.revisions, &revisionDiff{
		diff:       diff,
		merkleRoot: root,
		height:     height,
		ts:         ts,
	})
	ls.currentHeight = height
	ls.currentTs = ts

	// TODO - move this a separate goroutine to prevent addRevision from blocking on IO
	// TODO - consider blocking the maximum length of revisions - to prevent crashing in case of failed flushes
	evicted, err := ls.evictRevisions()
	if err != nil {
		log.Error(err)
	}
	return evicted, nil
}

func (ls *rollingRevisions) evictRevisions() ([]primitives.MerkleSha256, error) {
	var evicted []primitives.MerkleSha256
	for len(ls.revisions) > ls.transientRevisions {
		d := ls.revisions[0]
		err := ls.prst.Write(d.height, d.ts, d.merkleRoot, d.diff)
		if err != nil {
			return evicted, err
		}
		evicted = append(evicted, ls.prstRoot)
		ls.prstHight = d.height
		ls.prstTs = d.ts
		ls.prstRoot = d.merkleRoot
		ls.revisions = ls.revisions[1:]
	}
	return evicted, nil
}

func (ls *rollingRevisions) getRevisionRecord(height primitives.BlockHeight, contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	if ls.currentHeight < height {
		return nil, false, errors.Errorf("requested height %d is too new. most recent available block height is %d", height, ls.currentHeight)
	}

	for i := len(ls.revisions) - 1; i >= 0; i-- {
		if ls.revisions[i].height > height {
			continue
		}
		if record, exists := ls.revisions[i].diff[contract][key]; exists {
			return record, !isZeroValue(record.Value()), nil // cached state increments must include zero values
		}
	}

	if ls.prstHight > height {
		return nil, false, errors.Errorf("requested height %d is too old. oldest available block height is %d", height, ls.prstHight)
	}
	return ls.prst.Read(contract, key)
}

func (ls *rollingRevisions) getRevisionHash(height primitives.BlockHeight) (primitives.MerkleSha256, error) {
	for i := len(ls.revisions) - 1; i >= 0; i-- {
		if ls.revisions[i].height == height {
			return ls.revisions[i].merkleRoot, nil
		}
	}

	if height != ls.prstHight {
		return nil, fmt.Errorf("could not locate merkle hash for height %d. oldest available block height is %d", height, ls.prstHight)
	}

	return ls.prstRoot, nil
}

func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}
