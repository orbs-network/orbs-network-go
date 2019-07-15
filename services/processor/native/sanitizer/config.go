// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package sanitizer

import (
	"strings"
)

type SanitizerConfig struct {
	ImportWhitelist   map[string]string
	FunctionBlacklist map[string][]string
}

func (c SanitizerConfig) AllowedPrefixes() (prefixes []string) {
	for whitelist := range c.ImportWhitelist {
		if strings.HasSuffix(whitelist, `*"`) {
			prefixes = append(prefixes, whitelist[:len(whitelist)-2])
		}
	}

	return
}
