package adapter

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadStateWithNonExistingBlockHeight(t *testing.T) {
	d := NewInMemoryStatePersistence()
	_, err := d.ReadState(1, "foo")
	require.EqualError(t, err, "block 1 does not exist in snapshot history", "did not fail with error")
}

func TestReadStateWithNonExistingContractName(t *testing.T) {
	d := NewInMemoryStatePersistence()
	_, err := d.ReadState(0, "foo")
	require.EqualError(t, err, "contract foo does not exist", "did not fail with error")
}
