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

func parseNodesAndPeers(value interface{}) (nodes map[string]FederationNode, peers map[string]GossipPeer, err error) {
	nodes = make(map[string]FederationNode)
	peers = make(map[string]GossipPeer)

	if nodeList, ok := value.([]interface{}); ok {
		for _, item := range nodeList {
			kv := item.(map[string]interface{})

			if nodeAddress, err := hex.DecodeString(kv["address"].(string)); err != nil {
				return nodes, peers, err
			} else {
				nodeAddress := primitives.NodeAddress(nodeAddress)

				if i, err := parseUint32(kv["port"].(float64)); err != nil {
					return nodes, peers, err
				} else {
					gossipPort := int(i)

					nodes[nodeAddress.KeyForMap()] = &hardCodedFederationNode{
						nodeAddress: nodeAddress,
					}

					peers[nodeAddress.KeyForMap()] = &hardCodedGossipPeer{
						gossipEndpoint: kv["ip"].(string),
						gossipPort:     gossipPort,
					}
				}
			}
		}
	}

	return nodes, peers, nil
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

		if key == "constant-consensus-leader" {
			nodeAddress, err = hex.DecodeString(value.(string))
			cfg.SetConstantConsensusLeader(primitives.NodeAddress(nodeAddress))
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

		if key == "gossip-port" {
			var gossipPort uint32
			gossipPort, err = parseUint32(value.(float64))
			cfg.SetUint32(GOSSIP_LISTEN_PORT, gossipPort)
			continue
		}

		if key == "federation-nodes" {
			var nodes map[string]FederationNode
			var peers map[string]GossipPeer

			nodes, peers, err = parseNodesAndPeers(value)
			cfg.SetFederationNodes(nodes)
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
