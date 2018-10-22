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

type stateIncrement struct {
	diff       adapter.ChainState
	merkleRoot primitives.MerkleSha256
	height     primitives.BlockHeight
	ts         primitives.TimestampNano
}

type layeredState struct {
	persistence  adapter.StatePersistence
	maxLayers    int
	incCache     []*stateIncrement
	recentHeight primitives.BlockHeight
	pHeight      primitives.BlockHeight
	pRoot        primitives.MerkleSha256
	pTs          primitives.TimestampNano
}

func newLayeredState(p adapter.StatePersistence, maxLayers int) *layeredState {
	pHeight, pTs, pRoot, err := p.ReadMetadata()
	if err != nil {
		panic("could not load state metadata")
	}
	result := &layeredState{
		persistence:  p,
		maxLayers:    maxLayers,
		recentHeight: pHeight,
		pHeight:      pHeight,
		pTs:          pTs,
		pRoot:        pRoot,
	}

	return result
}

func (ls *layeredState) Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff adapter.ChainState) error {
	ls.incCache = append(ls.incCache, &stateIncrement{
		diff:       diff,
		merkleRoot: root,
		height:     height,
		ts:         ts,
	})
	ls.recentHeight = height
	// TODO - move this loop for merging and persisting snapshots to a separate goroutine. merely append here with a safety array size limit
	for len(ls.incCache) > ls.maxLayers {
		d := ls.incCache[0]
		err := ls.persistence.Write(d.height, d.ts, d.merkleRoot, d.diff)
		if err != nil {
			log.Error(err)
			break
		}
		ls.pHeight = d.height
		ls.pTs = d.ts
		ls.pRoot = d.merkleRoot
		ls.incCache = ls.incCache[1:]
	}
	return nil
}

func (ls *layeredState) Read(height primitives.BlockHeight, contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	if ls.recentHeight < height {
		return nil, false, errors.Errorf("requested height %d is too new. most recent available block height is %d", height, ls.recentHeight)
	}

	for i := len(ls.incCache) - 1; i >= 0; i-- {
		if ls.incCache[i].height > height {
			continue
		}
		if record, exists := ls.incCache[i].diff[contract][key]; exists {
			return record, !isZeroValue(record.Value()), nil // cached state increments must include zero values
		}
	}

	if ls.pHeight > height {
		return nil, false, errors.Errorf("requested height %d is too old. oldest available block height is %d", height, ls.pHeight)
	}
	return ls.persistence.Read(contract, key)
}

func (ls *layeredState) ReadStateHash(height primitives.BlockHeight) (primitives.MerkleSha256, error) {
	for i := len(ls.incCache) - 1; i >= 0; i-- {
		if ls.incCache[i].height == height {
			return ls.incCache[i].merkleRoot, nil
		}
	}

	if height != ls.pHeight {
		return nil, fmt.Errorf("could not locate merkle hash for height %d. oldest available block height is %d", height, ls.pHeight)
	}

	return ls.pRoot, nil
}

func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}
