// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"strconv"
	"strings"
	"time"
)

func newEmptyFileConfig(source string) (mutableNodeConfig, error) {
	return newFileConfig(emptyConfig(), source)
}

func newFileConfig(parent mutableNodeConfig, source string) (mutableNodeConfig, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(source), &data); err != nil {
		return nil, err
	}

	if err := populateConfig(parent, data); err != nil {
		return nil, err
	}

	return parent, nil
}

func convertKeyName(key string) string {
	return strings.ToUpper(strings.Replace(key, "-", "_", -1))
}

func parseUint32(f64 float64) (uint32, error) {
	s := fmt.Sprintf("%.0f", f64)
	if i, err := strconv.Atoi(s); err == nil {
		return uint32(i), nil
	} else {
		return 0, err
	}
}

func parseNodes(value interface{}) (nodes map[string]ValidatorNode, err error) {
	nodes = make(map[string]ValidatorNode)

	if nodeList, ok := value.([]interface{}); ok {
		for _, item := range nodeList {
			address := item.(string)

			if nodeAddress, err := hex.DecodeString(address); err != nil {
				return nodes, err
			} else {
				nodeAddress := primitives.NodeAddress(nodeAddress)

				nodes[nodeAddress.KeyForMap()] = &hardCodedValidatorNode{
					nodeAddress: nodeAddress,
				}
			}
		}
	}

	return nodes, nil
}

func parsePeers(value interface{}) (peers map[string]GossipPeer, err error) {
	peers = make(map[string]GossipPeer)

	if nodeList, ok := value.([]interface{}); ok {
		for _, item := range nodeList {
			kv := item.(map[string]interface{})

			if nodeAddress, err := hex.DecodeString(kv["address"].(string)); err != nil {
				return peers, err
			} else {
				nodeAddress := primitives.NodeAddress(nodeAddress)

				if i, err := parseUint32(kv["port"].(float64)); err != nil {
					return peers, err
				} else {
					gossipPort := int(i)

					peers[nodeAddress.KeyForMap()] = &hardCodedGossipPeer{
						gossipEndpoint: kv["ip"].(string),
						gossipPort:     gossipPort,
					}
				}
			}
		}
	}

	return peers, nil
}

func populateConfig(cfg mutableNodeConfig, data map[string]interface{}) error {
	for key, value := range data {
		var duration time.Duration
		var numericValue uint32
		var nodeAddress primitives.NodeAddress
		var stringValue string
		var err error

		switch value.(type) {
		case bool:
			cfg.SetBool(convertKeyName(key), value.(bool))
		case float64:
			numericValue, err = parseUint32(value.(float64))
		case string:
			// Sometimes we try to parse duration, but sometimes it's not worth it, like with Ethereum endpoint
			var decodeError error
			if duration, decodeError = time.ParseDuration(value.(string)); decodeError != nil {
				stringValue = value.(string)
			}
		}

		if numericValue != 0 {
			cfg.SetUint32(convertKeyName(key), numericValue)
		}

		if duration != 0 {
			cfg.SetDuration(convertKeyName(key), duration)
		}

		if key == "benchmark-consensus-constant-leader" {
			nodeAddress, err = hex.DecodeString(value.(string))
			cfg.SetBenchmarkConsensusConstantLeader(primitives.NodeAddress(nodeAddress))
			continue
		}

		if key == "active-consensus-algo" {
			var i uint32
			i, err = parseUint32(value.(float64))
			cfg.SetActiveConsensusAlgo(consensus.ConsensusAlgoType(i))
			continue
		}

		if key == "node-address" {
			nodeAddress, err = hex.DecodeString(value.(string))
			cfg.SetNodeAddress(primitives.NodeAddress(nodeAddress))
			continue
		}

		if key == "node-private-key" {
			var privateKey primitives.EcdsaSecp256K1PrivateKey
			privateKey, err = hex.DecodeString(value.(string))
			cfg.SetNodePrivateKey(primitives.EcdsaSecp256K1PrivateKey(privateKey))
			continue
		}

		if key == "ethereum-finality-blocks-component" {
			var finalityBlocksComponent uint32
			finalityBlocksComponent, err = parseUint32(value.(float64))
			cfg.SetUint32(ETHEREUM_FINALITY_BLOCKS_COMPONENT, finalityBlocksComponent)
			continue
		}

		if key == "gossip-port" {
			var gossipPort uint32
			gossipPort, err = parseUint32(value.(float64))
			cfg.SetUint32(GOSSIP_LISTEN_PORT, gossipPort)
			continue
		}

		if key == "genesis-validator-addresses" {
			var nodes map[string]ValidatorNode
			nodes, err = parseNodes(value)
			cfg.SetGenesisValidatorNodes(nodes)
			continue
		}

		if key == "federation-nodes" || key == "topology-nodes" { // note: "federation-nodes" is deprecated but kept for backwards-compatibility
			var peers map[string]GossipPeer
			peers, err = parsePeers(value)
			cfg.SetGossipPeers(peers)
			continue
		}

		if stringValue != "" {
			cfg.SetString(convertKeyName(key), stringValue)
		}

		if err != nil {
			return fmt.Errorf("could not decode value for config key %s: %s", key, err)
		}
	}

	return nil
}
