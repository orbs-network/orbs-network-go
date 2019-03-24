// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestToMemberIdsReturnsSameElementCount(t *testing.T) {
	nodeAddresses := keys.NodeAddressesForTests()
	count := len(nodeAddresses)
	memberIds := toMemberIds(nodeAddresses)
	require.Equal(t, count, len(memberIds), "toMemberIds() should return same count of memberIds as its input nodeAddresses")
}

func TestToMemberIdsReturnsEmptyWhenGivenNil(t *testing.T) {
	memberIds := toMemberIds(nil)
	require.Equal(t, 0, len(memberIds), "toMemberIds() should return empty memberIds when given nil")
}

func TestNodeAddressesToCommaSeparatedString_NilArray(t *testing.T) {
	require.Equal(t, "", nodeAddressesToCommaSeparatedString(nil))
}

func TestNodeAddressesToCommaSeparatedString(t *testing.T) {
	require.True(t, len(nodeAddressesToCommaSeparatedString(keys.NodeAddressesForTests())) > 0, "returned zero length string although input is not empty")
}
