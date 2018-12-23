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
