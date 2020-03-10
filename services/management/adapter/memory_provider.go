// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/management"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sort"
	"sync"
)

type MemoryConfig interface {
	GossipPeers() adapter.GossipPeers
	GenesisValidatorNodes() map[string]config.ValidatorNode
}

type ManagementProvider struct {
//	config MemoryConfig
	logger log.Logger

	sync.RWMutex
	currentReference uint64
	topology adapter.GossipPeers
	committees []*management.CommitteeTerm
}

func NewManagementProvider(config MemoryConfig, logger log.Logger) *ManagementProvider {
	committee := getCommitteeFromConfig(config)
	return  &ManagementProvider{currentReference: 0, topology: config.GossipPeers(), committees: []*management.CommitteeTerm{{0, committee}}, logger :logger}
}


func (mp *ManagementProvider) Update(ctx context.Context) (uint64, adapter.GossipPeers, []*management.CommitteeTerm, error) {
	mp.RLock()
	defer mp.RUnlock()

	return mp.currentReference, mp.topology, mp.committees, nil
}

func (mp *ManagementProvider) UpdateFromConfig(referenceNumber uint64, config MemoryConfig) error {
	mp.Lock()
	defer mp.Unlock()

	if mp.committees[len(mp.committees)-1].AsOfReference >= referenceNumber {
		return errors.Errorf("new committee must have an 'asOf' reference bigger than %d (and not %d)", mp.committees[len(mp.committees)-1].AsOfReference, referenceNumber)
	}

	mp.currentReference = referenceNumber
	mp.topology = make(adapter.GossipPeers)
	for key, peer := range config.GossipPeers() {
		mp.topology[key] = peer //copy input
	}
	mp.committees = append(mp.committees, &management.CommitteeTerm{ AsOfReference: referenceNumber, Committee: getCommitteeFromConfig(config)})

	// TODO NOAM log ?
	//mp.logger.Info("changing committee as Of block", log.Uint64("asOfReference", referenceNumber), log.StringableSlice("committee", committee))

	return nil
}

func getCommitteeFromConfig(config MemoryConfig) []primitives.NodeAddress {
	allNodes := config.GenesisValidatorNodes()
	var committee []primitives.NodeAddress

	for _, nodeAddress := range allNodes {
		committee = append(committee, nodeAddress.NodeAddress())
	}

	sort.SliceStable(committee, func(i, j int) bool {
		return bytes.Compare(committee[i], committee[j]) > 0
	})
	return committee
}


// TOOD NOAM find way to do testkeys
// TODO NOAM copy tests.
