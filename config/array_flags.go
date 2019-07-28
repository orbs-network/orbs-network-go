// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
)

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return "my string representation"
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func GetNodeConfigFromFiles(configFiles ArrayFlags, httpAddress string) (NodeConfig, error) {
	cfg := ForProduction("")

	if len(configFiles) != 0 {
		for _, configFile := range configFiles {
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				return nil, errors.Errorf("could not open MapBasedConfig file: %s", err)
			}

			contents, err := ioutil.ReadFile(configFile)
			if err != nil {
				return nil, err
			}

			cfg, err = cfg.MergeWithFileConfig(string(contents))

			if err != nil {
				return nil, err
			}
		}
	}

	cfg.SetString(HTTP_ADDRESS, httpAddress)

	return cfg, nil
}
