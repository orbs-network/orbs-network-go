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
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/management"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type FileConfig interface {
	VirtualChainId() primitives.VirtualChainId
	ManagementFilePath() string
	ManagementMaxFileSize() uint32
}

type FileProvider struct {
	logger log.Logger
	config FileConfig
	client *http.Client
}

func NewFileProvider(config FileConfig, logger log.Logger) *FileProvider {
	client := &http.Client{
		Timeout: 45 * time.Second,
	}
	return &FileProvider{config: config, logger: logger, client:client}
}

func (mp *FileProvider) Get(ctx context.Context, referenceTime primitives.TimestampSeconds) (*management.VirtualChainManagementData, error) {
	path := mp.generatePath(referenceTime)
	var contents []byte
	var err error

	if strings.HasPrefix(path, "http") {
		if contents, err = mp.readUrl(path); err != nil {
			mp.logger.Error("Provider url reading error", log.Error(err))
			return nil, err
		}
	} else {
		if contents, err = mp.readFile(path); err != nil {
			mp.logger.Error("Provider path file reading error", log.Error(err))
			return nil, err
		}
	}

	isHistoric := referenceTime != 0
	managementData, parseErr := mp.parseData(contents, isHistoric)
	if parseErr != nil {
		mp.logger.Error("Provider file parsing error", log.Error(parseErr))
		return nil, parseErr
	}

	return managementData, nil
}

func (mp *FileProvider) generatePath(referenceTime primitives.TimestampSeconds) string {
	var path string
	if referenceTime == 0 {
		path = mp.config.ManagementFilePath()
	} else {
		path = fmt.Sprintf("%s/%d", mp.config.ManagementFilePath(), referenceTime)
	}
	return path
}

func (mp *FileProvider) readUrl(path string) ([]byte, error) {
	res, err := mp.client.Get(path)
	defer res.Body.Close()

	if err != nil || res == nil {
		return nil, errors.Wrapf(err, "Failed http get of url %s", path)
	} else if res.ContentLength > 0 && uint32(res.ContentLength) > mp.config.ManagementMaxFileSize() { // TODO when no length given find other way ?
		return nil, errors.Wrapf(err, "Failed http get response too big %d", res.ContentLength)
	}

	return ioutil.ReadAll(res.Body)
}

func (mp *FileProvider) readFile(filePath string) ([]byte, error) {
	if fi, err := os.Stat(filePath); err != nil {
		return nil, errors.Errorf("could not open file: %s", err)
	} else if uint32(fi.Size()) > mp.config.ManagementMaxFileSize() {
		return nil, errors.Errorf("file too big (%d)", fi.Size())
	}

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file")
	}

	return contents, nil
}

type topologyNode struct {
	EthAddress  string
	OrbsAddress string
	Ip          string
	Port        int
}

type committee struct {
	EthAddress   string
	OrbsAddress  string
	Weight       uint64
	IdentityType uint
}

type committeeEvent struct {
	RefTime   uint64
	Committee []committee
}

type subscription struct {
	Status       string
	Tier         string
	RolloutGroup string
	IdentityType uint
}

type subscriptionEvent struct {
	RefTime uint64
	Data    subscription
}

type protocolVersion struct {
	RolloutGroup string
	Version      uint64
}

type protocolVersionEvent struct {
	RefTime uint64
	Data    protocolVersion
}

type vc struct {
	VirtualChainId        uint64
	GenesisRefTime        uint64
	CurrentTopology       []topologyNode
	CommitteeEvents       []committeeEvent
	SubscriptionEvents    []subscriptionEvent
	ProtocolVersionEvents []protocolVersionEvent
}

type mgmt struct {
	CurrentRefTime   uint64
	PageStartRefTime uint64
	PageEndRefTime   uint64
	VirtualChains    map[string]vc
}

func (mp *FileProvider) parseData(contents []byte, isHistoric bool) (*management.VirtualChainManagementData, error) {
	var data mgmt
	if err := json.Unmarshal(contents, &data); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal vcs data")
	}

	vcString := fmt.Sprintf("%d", mp.config.VirtualChainId())
	vcData, ok := data.VirtualChains[vcString]
	if !ok {
		return nil, errors.Errorf("could not find current vc in data (%d)", mp.config.VirtualChainId())
	}

	if data.CurrentRefTime != 0 {
		if isHistoric {
			if data.CurrentRefTime < data.PageEndRefTime || data.PageEndRefTime < data.PageStartRefTime {
				return nil, errors.Errorf("historic data : CurrentRefTime (%d) should be >= PageEndRefTime (%d) should be >= PageStartRefTime (%d)",
					data.CurrentRefTime, data.PageEndRefTime, data.PageStartRefTime)
			}
		} else {
			if data.CurrentRefTime != data.PageEndRefTime || data.PageEndRefTime < data.PageStartRefTime {
				return nil, errors.Errorf("data: CurrentRefTime (%d) should be equal to PageEndRefTime (%d) and it should be >= PageStartRefTime (%d)",
					data.CurrentRefTime, data.PageEndRefTime, data.PageStartRefTime)
			}

		}
	}

	topology, err := parseTopology(vcData.CurrentTopology)
	if err != nil {
		return nil, err
	}

	committeeTerms, err := parseCommittees(vcData.CommitteeEvents)
	if err != nil {
		return nil, err
	}

	subscriptions, err := parseSubscription(vcData.SubscriptionEvents)
	if err != nil {
		return nil, err
	}

	protocolVersions := parseProtocolVersion(vcData.ProtocolVersionEvents)

	return &management.VirtualChainManagementData{
		CurrentReference:   primitives.TimestampSeconds(data.CurrentRefTime),
		GenesisReference:   primitives.TimestampSeconds(vcData.GenesisRefTime),
		StartPageReference: primitives.TimestampSeconds(data.PageStartRefTime),
		EndPageReference:   primitives.TimestampSeconds(data.PageEndRefTime),
		CurrentTopology:    topology,
		Committees:         committeeTerms,
		Subscriptions:      subscriptions,
		ProtocolVersions:   protocolVersions,
	}, nil
}

func parseTopology(currentTopology []topologyNode) ([]*services.GossipPeer, error) {
	topology := make([]*services.GossipPeer, 0, len(currentTopology))
	for _, item := range currentTopology {
		hexAddress := item.OrbsAddress
		if nodeAddress, err := hex.DecodeString(hexAddress); err != nil {
			return nil, errors.Wrapf(err, "cannot translate topology node address from hex %s", hexAddress)
		} else if item.Ip == "" {
			return nil, errors.Errorf("empty ip address for node %s", hexAddress)
		} else if item.Port < 1024 || item.Port > 65535 {
			return nil, errors.Errorf("topology node port %d needs to be 1024-65535 range", item.Port)
		} else {
			nodeAddress := primitives.NodeAddress(nodeAddress)
			topology = append(topology, &services.GossipPeer{Address: nodeAddress, Endpoint: item.Ip, Port: uint32(item.Port)})
		}
	}
	return topology, nil
}

func parseCommittees(committeeEvents []committeeEvent) ([]management.CommitteeTerm, error) {
	var committeeTerms []management.CommitteeTerm
	for _, event := range committeeEvents {
		var committee []primitives.NodeAddress
		var weights   []primitives.Weight

		for _, member := range event.Committee {
			if address, err := hex.DecodeString(member.OrbsAddress); err != nil {
				return nil, errors.Wrapf(err, "cannot decode committee node address hex %s", address)
			} else if member.Weight == 0 {
				return nil, errors.Errorf("Weight of node %s is 0 or missing", address)
			} else {
				committee = append(committee, primitives.NodeAddress(address))
				weights = append(weights, primitives.Weight(member.Weight))
			}
		}

		sort.SliceStable(committee, func(i, j int) bool {
			return bytes.Compare(committee[i], committee[j]) > 0
		})

		committeeTerms = append(committeeTerms, management.CommitteeTerm{AsOfReference: primitives.TimestampSeconds(event.RefTime), Members: committee, Weights: weights})
	}

	return committeeTerms, nil
}

func parseSubscription(subscriptionEvents []subscriptionEvent) ([]management.SubscriptionTerm, error) {
	if len(subscriptionEvents) == 0 {
		return nil, errors.New("cannot start virtual chain with no subscription data.")
	}

	var subscriptionPeriods []management.SubscriptionTerm
	for _, event := range subscriptionEvents {
		isActive := false
		if event.Data.Status == "active" {
			isActive = true
		}
		subscriptionPeriods = append(subscriptionPeriods, management.SubscriptionTerm{AsOfReference: primitives.TimestampSeconds(event.RefTime), IsActive: isActive})
	}

	return subscriptionPeriods, nil
}

func parseProtocolVersion(protocolVersionEvents []protocolVersionEvent) []management.ProtocolVersionTerm {
	var protocolVersionPeriods []management.ProtocolVersionTerm
	for _, event := range protocolVersionEvents {
		protocolVersionPeriods = append(protocolVersionPeriods, management.ProtocolVersionTerm{AsOfReference: primitives.TimestampSeconds(event.RefTime), Version: primitives.ProtocolVersion(event.Data.Version)})
	}

	if len(protocolVersionPeriods) == 0 {
		protocolVersionPeriods = append(protocolVersionPeriods, management.ProtocolVersionTerm{AsOfReference: 0, Version: config.MINIMAL_CONSENSUS_BLOCK_PROTOCOL_VERSION})
	}

	// TODO POSV2 consider if last PV is larger than config.maximalpv -> fail ?

	return protocolVersionPeriods
}
