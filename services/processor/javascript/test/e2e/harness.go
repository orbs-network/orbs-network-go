// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
// +build javascript

package e2e

import (
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/crypto-lib-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/e2e"
	"github.com/stretchr/testify/require"
	"testing"
)

func DeployJSContractAndRequireSuccess(h *e2e.Harness, t *testing.T, keyPair *keys.Ed25519KeyPair, contractName string, contractBytes ...[]byte) {

	h.WaitUntilTransactionPoolIsReady(t)

	dcExResult, dcErr := DeployJSContract(h, keyPair, contractName, contractBytes...)

	require.Nil(t, dcErr, "expected deploy contract to succeed")
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS, dcExResult.ExecutionResult, "expected deploy contract to succeed")
}

func DeployJSContract(h *e2e.Harness, from *keys.Ed25519KeyPair, contractName string, code ...[]byte) (*codec.TransactionResponse, error) {
	return h.DeployContract(from, contractName, orbsClient.PROCESSOR_TYPE_JAVASCRIPT, code...)
}
