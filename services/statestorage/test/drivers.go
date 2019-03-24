// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type Driver struct {
	service services.StateStorage
}

type keyValue struct {
	key   string
	value []byte
}

func NewStateStorageDriver(numOfStateRevisionsToRetain uint32) *Driver {
	return newStateStorageDriverWithGrace(numOfStateRevisionsToRetain, 0, 0)
}

func newStateStorageDriverWithGrace(numOfStateRevisionsToRetain uint32, graceBlockDiff uint32, graceTimeoutMillis uint64) *Driver {
	if numOfStateRevisionsToRetain <= 0 {
		numOfStateRevisionsToRetain = 1
	}

	cfg := config.ForStateStorageTest(numOfStateRevisionsToRetain, graceBlockDiff, graceTimeoutMillis)
	registry := metric.NewRegistry()

	p := memory.NewStatePersistence(registry)
	logger := log.GetLogger().WithOutput() // a mute logger

	return &Driver{service: statestorage.NewStateStorage(cfg, p, nil, logger, registry)}
}

func (d *Driver) ReadSingleKey(ctx context.Context, contract string, key string) ([]byte, error) {
	h, _, _ := d.GetBlockHeightAndTimestamp(ctx)
	return d.ReadSingleKeyFromRevision(ctx, h, contract, key)
}

func (d *Driver) ReadSingleKeyFromRevision(ctx context.Context, revision int, contract string, key string) ([]byte, error) {
	out, err := d.ReadKeysFromRevision(ctx, revision, contract, key)
	if err != nil {
		return nil, err
	}
	return out[0].value, nil
}

func (d *Driver) ReadKeys(ctx context.Context, contract string, keys ...string) ([]*keyValue, error) {
	h, _, _ := d.GetBlockHeightAndTimestamp(ctx)
	return d.ReadKeysFromRevision(ctx, h, contract, keys...)
}

func (d *Driver) ReadKeysFromRevision(ctx context.Context, revision int, contract string, keys ...string) ([]*keyValue, error) {
	ripmdKeys := make([][]byte, 0, len(keys))
	for _, key := range keys {
		ripmdKeys = append(ripmdKeys, []byte(key))
	}
	out, err := d.service.ReadKeys(ctx, &services.ReadKeysInput{BlockHeight: primitives.BlockHeight(revision), ContractName: primitives.ContractName(contract), Keys: ripmdKeys})

	if err != nil {
		return nil, err
	}

	if l, k := len(out.StateRecords), len(keys); l != k {
		panic(fmt.Sprintf("expected exactly %v elements in array. found %v", k, l))
	}

	result := make([]*keyValue, 0, len(keys))
	for i := range out.StateRecords {
		if actual, expected := out.StateRecords[i].Key(), keys[i]; !bytes.Equal(actual, []byte(expected)) {
			panic(fmt.Sprintf("expected output key %s to match input key %s", actual, expected))
		}
		result = append(result, &keyValue{string(out.StateRecords[i].Key()), out.StateRecords[i].Value()})
	}

	return result, nil
}

func (d *Driver) GetBlockHeightAndTimestamp(ctx context.Context) (int, int, error) {
	output, err := d.service.GetStateStorageBlockHeight(ctx, &services.GetStateStorageBlockHeightInput{})
	return int(output.LastCommittedBlockHeight), int(output.LastCommittedBlockTimestamp), err
}

func (d *Driver) CommitStateDiff(ctx context.Context, state *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	return d.service.CommitStateDiff(ctx, state)
}

func (d *Driver) CommitValuePairs(ctx context.Context, contract string, keyValues ...string) {
	h, _, _ := d.GetBlockHeightAndTimestamp(ctx)
	d.CommitValuePairsAtHeight(ctx, h+1, contract, keyValues...)
}

func (d *Driver) CommitValuePairsAtHeight(ctx context.Context, h int, contract string, keyValues ...string) (*services.CommitStateDiffOutput, error) {
	if len(keyValues)%2 != 0 {
		panic("expecting an array of key value pairs")
	}
	b := builders.ContractStateDiff().WithContractName(contract)

	for i := 0; i < len(keyValues); i += 2 {
		b.WithStringRecord(keyValues[i], keyValues[i+1])
	}

	contractStateDiff := b.Build()
	return d.CommitStateDiff(ctx, CommitStateDiff().WithBlockHeight(int(h)).WithDiff(contractStateDiff).Build())
}
