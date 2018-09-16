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

func NewFileConfig(source string) (NodeConfig, error) {
	cfg := EmptyConfig()

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(source), &data); err != nil {
		return nil, err
	}

	if err := populateConfig(cfg, data); err != nil {
		return nil, err
	}

	return cfg, nil
}

func convertKeyName(key string) string {
	return strings.ToUpper(strings.Replace(key, "-", "_", -1))
}

func populateConfig(cfg NodeConfig, data map[string]interface{}) error {
	for key, value := range data {
		var duration time.Duration
		var numericValue uint32

		switch value.(type) {
		case float64:
			f64 := value.(float64)

			s := fmt.Sprintf("%.0f", f64)
			if i, err := strconv.Atoi(s); err == nil {
				numericValue = uint32(i)
			} else {
				return fmt.Errorf("could not decode value for config key %s: %s", key, err)
			}
		case string:
			s := value.(string)

			if parsedDuration, err := time.ParseDuration(s); err == nil {
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
			if publicKey, err := hex.DecodeString(value.(string)); err == nil {
				cfg.SetConstantConsensusLeader(primitives.Ed25519PublicKey(publicKey))
			} else {
				return fmt.Errorf("could not decode value for config key %s: %s", key, err)
			}
		}

		if key == "active-consensus-algo" {
			s := fmt.Sprintf("%.0f", value)
			if i, err := strconv.Atoi(s); err == nil {
				cfg.SetActiveConsensusAlgo(consensus.ConsensusAlgoType(i))
			} else {
				return fmt.Errorf("could not decode value for config key %s: %s", key, err)
			}
		}

		if key == "node-public-key" {
			if publicKey, err := hex.DecodeString(value.(string)); err == nil {
				cfg.SetNodePublicKey(primitives.Ed25519PublicKey(publicKey))
			} else {
				return fmt.Errorf("could not decode value for config key %s: %s", key, err)
			}
		}

		if key == "node-private-key" {
			if publicKey, err := hex.DecodeString(value.(string)); err == nil {
				cfg.SetNodePrivateKey(primitives.Ed25519PrivateKey(publicKey))
			} else {
				return fmt.Errorf("could not decode value for config key %s: %s", key, err)
			}
		}

		if key == "federation-nodes" {
			nodes := make(map[string]FederationNode)

			if nodeList, ok := value.([]interface{}); ok {
				for _, item := range nodeList {
					kv := item.(map[string]interface{})

					if publicKey, err := hex.DecodeString(kv["Key"].(string)); err == nil {
						nodePublicKey := primitives.Ed25519PublicKey(publicKey)

						var gossipPort uint16

						s := fmt.Sprintf("%.0f", kv["Port"])
						if i, err := strconv.Atoi(s); err == nil {
							gossipPort = uint16(i)
						} else {
							return fmt.Errorf("could not decode value for config key %s: %s", key, err)
						}

						nodes[nodePublicKey.String()] = &hardCodedFederationNode{
							nodePublicKey:  nodePublicKey,
							gossipEndpoint: kv["IP"].(string),
							gossipPort:     gossipPort,
						}
					} else {
						return fmt.Errorf("could not decode value for config key %s: %s", key, err)
					}

				}

				cfg.SetFederationNodes(nodes)
			} else {
				return fmt.Errorf("could not decode value for config key %s", key)
			}
		}
	}

	return nil
}
