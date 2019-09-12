// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer"
)

func (s *service) sanitizeDeployedSourceCode(code string) (string, error) {
	if s.config.ProcessorSanitizeDeployedContracts() {
		return s.sanitizer.Process(code)
	} else {
		return code, nil
	}
}

func (s *service) createSanitizer() *sanitizer.Sanitizer {
	return sanitizer.NewSanitizer(SanitizerConfigForProduction())
}

func SanitizerConfigForProduction() *sanitizer.SanitizerConfig {
	return &sanitizer.SanitizerConfig{
		ImportWhitelist: map[string]string{
			// package: reason to whitelist

			// SDK
			`"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/*"`: "SDK",

			// Contract external libraries
			`"github.com/orbs-network/contract-external-libraries-go/v1/*"`: "Contract external libraries",

			// Text
			`"strings"`:       "Text manipulation",
			`"strconv"`:       "Text manipulation",
			`"text/template"`: "Text manipulation",

			// Time
			`"time"`: "Time manipulation",

			// Binary
			`"bytes"`:           "Binary manipulation",
			`"encoding/binary"`: "Binary manipulation",
			`"io"`:              "Binary manipulation",

			// Encoding
			`"encoding/json"`:   "Serialization",
			`"encoding/hex"`:    "Serialization",
			`"encoding/base32"`: "Serialization",
			`"encoding/base64"`: "Serialization",

			// Utils
			`"sort"`: "Sorting collections of primitives",

			// Crypto
			`"hash"`:                        "Crypto",
			`"crypto"`:                      "Crypto",
			`"crypto/*"`:                    "Crypto",
			`"golang.org/x/crypto"`:         "Crypto",
			`"golang.org/x/crypto/ed25519"`: "ED25519",
			`"golang.org/x/crypto/sha3"`:    "SHA-3",
		},
		FunctionBlacklist: map[string][]string{
			"time": {
				"After",
				"AfterFunc",
				"Sleep",
				"Tick",
				"NewTimer",
				"NewTicker",
				"Now",
			},
		},
	}
}
