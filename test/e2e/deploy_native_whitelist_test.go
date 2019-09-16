//+build !race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContractWhitelist(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := newAppHarness()
		lt := time.Now()
		printTestTime(t, "started", &lt)

		h.waitUntilTransactionPoolIsReady(t)
		printTestTime(t, "first block committed", &lt)

		counterStart := uint64(time.Now().UnixNano())
		contractName := fmt.Sprintf("Whitelist%d", counterStart)
		contractSource, _ := ioutil.ReadFile("../contracts/whitelist/whitelist.go")

		printTestTime(t, "send deploy - start", &lt)

		h.deployContractAndRequireSuccess(t, OwnerOfAllSupply, contractName,
			[]byte(contractSource))

		printTestTime(t, "send deploy - end", &lt)

		sha2Response, err := h.runQuery(OwnerOfAllSupply.PublicKey(), contractName, "sha2_256", []byte(contractName))
		require.NoError(t, err)
		sha2ExpectedValue := sha256.Sum256([]byte(contractName))
		require.EqualValues(t, sha2ExpectedValue[:], sha2Response.OutputArguments[0])

		sha3Response, err := h.runQuery(OwnerOfAllSupply.PublicKey(), contractName, "sha3_256", []byte(contractName))
		require.NoError(t, err)
		sha3ExpectedValue := sha3.Sum256([]byte(contractName))
		require.EqualValues(t, sha3ExpectedValue[:], sha3Response.OutputArguments[0])
	})
}
