package statestorage

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type merkleRevisions interface {
	Update(rootMerkle primitives.MerkleSha256, diffs merkle.MerkleDiffs) (primitives.MerkleSha256, error)
	Forget(rootHash primitives.MerkleSha256)
}

type revisionDiff struct {
	diff       adapter.ChainState
	merkleRoot primitives.MerkleSha256
	height     primitives.BlockHeight
	ts         primitives.TimestampNano
}

type rollingRevisions struct {
	persist            adapter.StatePersistence
	transientRevisions int
	revisions          []*revisionDiff
	merkle             merkleRevisions
	currentHeight      primitives.BlockHeight
	currentTs          primitives.TimestampNano
	currentMerkleRoot  primitives.MerkleSha256
	persistedHeight    primitives.BlockHeight
	persistedRoot      primitives.MerkleSha256
	persistedTs        primitives.TimestampNano
}

func newRollingRevisions(persist adapter.StatePersistence, transientRevisions int, merkle merkleRevisions) *rollingRevisions {
	h, ts, r, err := persist.ReadMetadata()
	if err != nil {
		panic("could not load state metadata")
	}

	result := &rollingRevisions{
		persist:            persist,
		transientRevisions: transientRevisions,
		merkle:             merkle,
		currentHeight:      h,
		currentTs:          ts,
		currentMerkleRoot:  r,
		persistedHeight:    h,
		persistedTs:        ts,
		persistedRoot:      r,
	}

	return result
}

func (ls *rollingRevisions) getCurrentHeight() primitives.BlockHeight {
	return ls.currentHeight
}

func (ls *rollingRevisions) getCurrentTimestamp() primitives.TimestampNano {
	return ls.currentTs
}

func (ls *rollingRevisions) addRevision(height primitives.BlockHeight, ts primitives.TimestampNano, diff adapter.ChainState) error {
	newRoot, err := ls.merkle.Update(ls.currentMerkleRoot, filterToMerkleInput(diff))
	if err != nil {
		return errors.Wrapf(err, "failed to updated merkle tree")
	}

	ls.revisions = append(ls.revisions, &revisionDiff{
		diff:       diff,
		merkleRoot: newRoot,
		height:     height,
		ts:         ts,
	})
	ls.currentHeight = height
	ls.currentTs = ts
	ls.currentMerkleRoot = newRoot

	// TODO - move this a separate goroutine to prevent addRevision from blocking on IO
	// TODO - consider blocking the maximum length of revisions - to prevent crashing in case of failed flushes
	return ls.evictRevisions()
}

func filterToMerkleInput(diff adapter.ChainState) merkle.MerkleDiffs {
	result := make(merkle.MerkleDiffs)
	for contractName, contractState := range diff {
		for _, r := range contractState {
			k, v := getMerkleEntry(contractName, r.Key(), r.Value())
			result[k] = v
		}
	}
	return result
}

func getMerkleEntry(contract primitives.ContractName, key primitives.Ripmd160Sha256, value []byte) (string, primitives.Sha256) {
	return string(hash.CalcSha256(append([]byte(contract), key...))), hash.CalcSha256(value)
}

func (ls *rollingRevisions) evictRevisions() error {
	for len(ls.revisions) > ls.transientRevisions {
		d := ls.revisions[0]
		err := ls.persist.Write(d.height, d.ts, d.merkleRoot, d.diff)
		if err != nil {
			return err
		}
		ls.merkle.Forget(ls.persistedRoot)

		ls.persistedHeight = d.height
		ls.persistedTs = d.ts
		ls.persistedRoot = d.merkleRoot
		ls.revisions = ls.revisions[1:]
	}
	return nil
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

	if ls.persistedHeight > height {
		return nil, false, errors.Errorf("requested height %d is too old. oldest available block height is %d", height, ls.persistedHeight)
	}
	return ls.persist.Read(contract, key)
}

func (ls *rollingRevisions) getRevisionHash(height primitives.BlockHeight) (primitives.MerkleSha256, error) {
	for i := len(ls.revisions) - 1; i >= 0; i-- {
		if ls.revisions[i].height == height {
			return ls.revisions[i].merkleRoot, nil
		}
	}

	if height != ls.persistedHeight {
		return nil, fmt.Errorf("could not locate merkle hash for height %d. oldest available block height is %d", height, ls.persistedHeight)
	}

	return ls.persistedRoot, nil
}

func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}
