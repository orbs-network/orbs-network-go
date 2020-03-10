// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/management"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
)

type FileConfig interface {
	VirtualChainId() primitives.VirtualChainId
	ManagementFilePath() string
}

type FileProvider struct {
	logger log.Logger
	config FileConfig
}

func NewFileProvider(config FileConfig, logger log.Logger) *FileProvider {
	return &FileProvider{config: config, logger :logger}
}

func (mp *FileProvider) Update(ctx context.Context) (uint64, adapter.GossipPeers, []*management.CommitteeTerm, error) {
	path := mp.config.ManagementFilePath()
	var contents []byte
	var err error

	if strings.HasPrefix(path, "http") {
		if contents, err = readUrl(path); err != nil {
			mp.logger.Error("Provider url reading error", log.Error(err))
			return 0, nil, nil, err
		}
	} else {
		if contents, err = readFile(path); err != nil {
			mp.logger.Error("Provider path file reading error", log.Error(err))
			return 0, nil, nil, err
		}
	}

	reference, peers, committees, err := mp.parseFile(contents)
	if err != nil {
		mp.logger.Error("Provider file parsing error", log.Error(err))
		return 0, nil, nil, err
	}

	return reference, peers, committees, nil
}

func readUrl(path string) ([]byte, error) {
	res, err := http.Get(path)

	if err != nil || res == nil {
		return nil, errors.Errorf("")
	}
	return ioutil.ReadAll(res.Body)
}

func readFile(filePath string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, errors.Errorf("could not open file: %s", err)
	}

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file")
	}

	return contents, nil
}

type topologyNode struct {
	Address string
	Ip string
	Port int
}

type committee struct {
	EthAddress string
	OrbsAddress string
	EffectiveStake uint64
	IdentityType uint
}

type committeeEvent struct {
	RefTime uint64
	Committee []committee
}

type vc struct {
	CurrentTopology []topologyNode
	CommitteeEvents []committeeEvent
}

type mgmt struct {
	CurrentRefTime uint64
	PageStartRefTime uint64
	PageEndRefTime uint64
	VirtualChains map[string]vc
}

func (mp *FileProvider) parseFile(contents []byte) (uint64, adapter.GossipPeers, []*management.CommitteeTerm, error) {
	var data mgmt
	if err := json.Unmarshal(contents, &data); err != nil {
		return 0, nil, nil, errors.Wrapf(err, "could not unmarshal vcs data")
	}

	vcString := fmt.Sprintf("%d", mp.config.VirtualChainId())
	vcData, ok := data.VirtualChains[vcString]
	if !ok {
		return 0, nil, nil, errors.Errorf("could not find current vc in data")
	}

	peers := make(adapter.GossipPeers)
	for _, item := range vcData.CurrentTopology {
		hexAddress := item.Address
		if nodeAddress, err := hex.DecodeString(hexAddress); err != nil {
			return 0, nil, nil, errors.Wrapf(err, "cannot decode topology node address hex %s", hexAddress)
		} else {
			nodeAddress := primitives.NodeAddress(nodeAddress)
			peers[nodeAddress.KeyForMap()] = adapter.NewGossipPeer(item.Port, item.Ip, hexAddress)
		}
	}

	var committeeTerms []*management.CommitteeTerm
	for _, committeeEvent := range vcData.CommitteeEvents {
		var committee []primitives.NodeAddress

		for _, nodeAddress := range committeeEvent.Committee {
			if address, err := hex.DecodeString(nodeAddress.OrbsAddress); err != nil {
				return 0, nil, nil, errors.Wrapf(err, "cannot decode committee node address hex %s", address)
			} else {
				committee = append(committee, primitives.NodeAddress(address))
			}
		}

		sort.SliceStable(committee, func(i, j int) bool {
			return bytes.Compare(committee[i], committee[j]) > 0
		})

		committeeTerms = append(committeeTerms, &management.CommitteeTerm{AsOfReference:committeeEvent.RefTime, Committee: committee})
	}


	return data.CurrentRefTime, peers, committeeTerms, nil
}
