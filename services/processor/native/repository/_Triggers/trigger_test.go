// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
package triggers_systemcontract

import (
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsCommitteeContract_updateMisses_SignerExistsPanics(t *testing.T) {
	signerAddress := AnAddress()

	InServiceScope(signerAddress, nil, func(m Mockery) {
		_init()

		// run & assert
		require.Panics(t, func() {
			trigger()
		}, "should panic because a signer exists")
	})
}
