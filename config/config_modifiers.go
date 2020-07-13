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
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// Mutate
func (c *config) Modify(newValues ...NodeConfigKeyValue) {
	for _, kv := range newValues {
		c.kv[kv.Key] = kv.Value
	}
}

func modifyFromJson(cfg mutableNodeConfig, source string) error {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(source), &data); err != nil {
		return err
	}

	if err := populateConfig(cfg, data); err != nil {
		return err
	}

	return nil
}

func convertKeyName(key string) string {
	return strings.ToUpper(strings.Replace(key, "-", "_", -1))
}

func populateConfig(cfg mutableNodeConfig, data map[string]interface{}) error {
	for key, value := range data {

		if key == "benchmark-consensus-constant-leader" {
			nodeAddress, err := hex.DecodeString(value.(string))
			if err != nil {
				return fmt.Errorf("could not decode value for config key %s: %s", key, err)
			}
			cfg.SetBenchmarkConsensusConstantLeader(primitives.NodeAddress(nodeAddress))
			continue
		} else if key == "active-consensus-algo" {
			cfg.SetActiveConsensusAlgo(consensus.ConsensusAlgoType(value.(float64)))
			continue
		} else if key == "node-address" {
			nodeAddress, err := hex.DecodeString(value.(string))
			if err != nil {
				return fmt.Errorf("could not decode value for config key %s: %s", key, err)
			}
			cfg.SetNodeAddress(nodeAddress)
			continue
		} else if key == "node-private-key" {
			var privateKey primitives.EcdsaSecp256K1PrivateKey
			privateKey, err := hex.DecodeString(value.(string))
			if err != nil {
				return fmt.Errorf("could not decode value for config key %s: %s", key, err)
			}
			cfg.SetNodePrivateKey(privateKey)
			continue
		}

		switch value.(type) {
		case bool:
			cfg.SetBool(convertKeyName(key), value.(bool))
		case float64:
			cfg.SetUint32(convertKeyName(key), uint32(value.(float64)))
		case string:
			if duration, decodeError := time.ParseDuration(value.(string)); decodeError != nil {
				cfg.SetString(convertKeyName(key), value.(string))
			} else {
				cfg.SetDuration(convertKeyName(key), duration)
			}
		}
	}

	return nil
}

// For main reading several files into one config

type FilesPaths []string

func (i *FilesPaths) String() string {
	return "my string representation"
}

func (i *FilesPaths) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func GetNodeConfigFromFiles(configFiles FilesPaths, httpAddress string) (NodeConfig, error) {
	cfg := ForProduction("")

	if len(configFiles) != 0 {
		for _, configFile := range configFiles {
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				return nil, errors.Errorf("could not open config file: %s", err)
			}

			contents, err := ioutil.ReadFile(configFile)
			if err != nil {
				return nil, err
			}

			err = modifyFromJson(cfg, string(contents))

			if err != nil {
				return nil, err
			}
		}
	}

	cfg.SetString(HTTP_ADDRESS, httpAddress)

	return cfg, nil
}
