package config

import (
	"github.com/orbs-network/orbs-network-go/test/files"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewFromMultipleFiles_HandlesEmptyFileList(t *testing.T) {
	_, err := NewFromMultipleFiles()
	require.NoError(t, err, "failed parsing config file")
}

func TestNewFromMultipleFiles_MergesPropsFromBothFiles(t *testing.T) {
	f1 := files.NewTempFileWithContent(t, `{"a":"b"}`)
	defer files.RemoveSilently(f1)
	f2 := files.NewTempFileWithContent(t, `{"c":"d"}`)
	defer files.RemoveSilently(f2)

	cfg, err := NewFromMultipleFiles(f1, f2)
	require.NoError(t, err, "failed reading config files")

	require.Equal(t, "b", cfg.kv["A"].StringValue)
	require.Equal(t, "d", cfg.kv["C"].StringValue)
}

func TestNewFromMultipleFiles_OverridesCommonPropsAccordingToFileNameOrder(t *testing.T) {
	f1 := files.NewTempFileWithContent(t, `{"a":"b"}`)
	defer files.RemoveSilently(f1)
	f2 := files.NewTempFileWithContent(t, `{"a":"c"}`)
	defer files.RemoveSilently(f2)

	cfg, err := NewFromMultipleFiles(f1, f2)
	require.NoError(t, err, "failed reading config files")

	require.Equal(t, "c", cfg.kv["A"].StringValue, "later config file did not override earlier config file's prop")
}

func TestNewFromMultipleFiles_ForE2E(t *testing.T) {
	cfg, err := NewFromMultipleFiles("../docker/test/benchmark-config/node1.json")
	require.NoError(t, err, "failed parsing config file")

	require.EqualValues(t, "a328846cd5b4979d68a8c58a9bdfeee657b34de7", cfg.NodeAddress().String())
}
