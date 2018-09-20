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

func NewEmptyFileConfig(source string) (MutableNodeConfig, error) {
	return NewFileConfig(EmptyConfig(), source)
}

func NewFileConfig(parent MutableNodeConfig, source string) (MutableNodeConfig, error) {
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

// TODO notify about ignored entries?
func parseNodeList(value interface{}) (nodes map[string]FederationNode, err error) {
	nodes = make(map[string]FederationNode)

	if nodeList, ok := value.([]interface{}); ok {
		for _, item := range nodeList {
			kv := item.(map[string]interface{})

			if publicKey, err := hex.DecodeString(kv["Key"].(string)); err == nil {
				nodePublicKey := primitives.Ed25519PublicKey(publicKey)

				var gossipPort uint16
				var i uint32

				if i, err = parseUint32(kv["Port"].(float64)); err == nil {
					gossipPort = uint16(i)

					nodes[nodePublicKey.String()] = &hardCodedFederationNode{
						nodePublicKey:  nodePublicKey,
						gossipEndpoint: kv["IP"].(string),
						gossipPort:     gossipPort,
					}
				}
			}
		}
	}

	return nodes, err
}

func populateConfig(cfg MutableNodeConfig, data map[string]interface{}) (err error) {
	for key, value := range data {
		var duration time.Duration
		var numericValue uint32
		var publicKey primitives.Ed25519PublicKey

		switch value.(type) {
		case float64:
			numericValue, err = parseUint32(value.(float64))
		case string:
			if parsedDuration, err := time.ParseDuration(value.(string)); err == nil {
				duration = parsedDuration
			}
		}

		if numericValue != 0 {
			cfg.SetUint32(convertKeyName(key), numericValue)
		}

		if duration != 0 {
			cfg.SetDuration(convertKeyName(key), duration)
		}

		if key == "constant-consensus-leader" {
			publicKey, err = hex.DecodeString(value.(string))
			cfg.SetConstantConsensusLeader(primitives.Ed25519PublicKey(publicKey))
		}

		if key == "active-consensus-algo" {
			var i uint32
			i, err = parseUint32(value.(float64))
			cfg.SetActiveConsensusAlgo(consensus.ConsensusAlgoType(i))
		}

		if key == "node-public-key" {
			publicKey, err = hex.DecodeString(value.(string))
			cfg.SetNodePublicKey(primitives.Ed25519PublicKey(publicKey))
		}

		if key == "node-private-key" {
			var privateKey primitives.Ed25519PrivateKey
			privateKey, err = hex.DecodeString(value.(string))
			cfg.SetNodePrivateKey(primitives.Ed25519PrivateKey(privateKey))
		}

		if key == "federation-nodes" {
			var nodes map[string]FederationNode
			nodes, err = parseNodeList(value)
			cfg.SetFederationNodes(nodes)
		}

		if err != nil {
			return fmt.Errorf("could not decode value for config key %s: %s", key, err)
		}
	}

	return nil
}
