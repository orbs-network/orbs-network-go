// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package file

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
)

type Config interface {
	VirtualChainId() primitives.VirtualChainId
	GossipTopologyFilePath() string
}

type TopologyProvider struct {
	logger log.Logger
	config Config
	sync.RWMutex
	topology adapter.GossipPeers
}

func NewTopologyProvider(config Config, logger log.Logger) *TopologyProvider {
	return &TopologyProvider{config: config, logger :logger}
}

func (tp *TopologyProvider) GetTopology(ctx context.Context) adapter.GossipPeers {
	tp.RLock()
	defer tp.RUnlock()
	return tp.topology
}

func (tp *TopologyProvider) UpdateTopology(ctx context.Context) error {
	path := tp.config.GossipTopologyFilePath()
	var contents []byte
	var err error

	if strings.HasPrefix(path, "http") {
		if contents, err = readUrl(path); err != nil {
			tp.logger.Error("TopologyProvider url reading error", log.Error(err))
			return err
		}
	} else {
		if contents, err = readFile(path); err != nil {
			tp.logger.Error("TopologyProvider file reading error", log.Error(err))
			return err
		}
	}

	peers, errPeers := tp.parseFile(contents)
	if errPeers != nil {
		tp.logger.Error("TopologyProvider file parsing error", log.Error(err))
		return errPeers
	}
	tp.writeTopology(peers)
	return nil
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

type vc struct {
	TopologyNodes []topologyNode `json:"topology-nodes"`
}

func (tp *TopologyProvider) parseFile(contents []byte) (adapter.GossipPeers, error) {
	var data map[string]vc
	if err := json.Unmarshal(contents, &data); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal vcs data")
	}

	vcString := fmt.Sprintf("%d", tp.config.VirtualChainId())
	vcData, ok := data[vcString]
	if !ok {
		return nil, errors.Errorf("could not find current vc in data")
	}

	peers := make(adapter.GossipPeers)
	for _, item := range vcData.TopologyNodes {
		hexAddress := item.Address
		if nodeAddress, err := hex.DecodeString(hexAddress); err != nil {
			return nil, errors.Wrapf(err, "cannot decode node address hex %s", hexAddress)
		} else {
			nodeAddress := primitives.NodeAddress(nodeAddress)
			peers[nodeAddress.KeyForMap()] = adapter.NewGossipPeer(item.Port, item.Ip, hexAddress)
		}
	}

	return peers, nil
}

func (tp *TopologyProvider) writeTopology(peers adapter.GossipPeers) {
	tp.Lock()
	defer tp.Unlock()
	tp.topology = peers
}
