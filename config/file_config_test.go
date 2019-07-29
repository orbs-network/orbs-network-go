package config

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/files"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
	"time"
)

func TestNewFromMultipleFiles_HandlesEmptyFileList(t *testing.T) {
	_, err := NewLoader().Load()
	require.NoError(t, err, "failed parsing config file")
}

func TestNewFromMultipleFiles_MergesPropsFromBothFiles(t *testing.T) {
	f1 := files.NewTempFileWithContent(t, `{"a":"b"}`)
	defer files.RemoveSilently(f1)
	f2 := files.NewTempFileWithContent(t, `{"c":"d"}`)
	defer files.RemoveSilently(f2)

	cfg, err := NewLoader(f1, f2).Load()
	require.NoError(t, err, "failed reading config files")

	require.Equal(t, "b", cfg.kv["A"].StringValue)
	require.Equal(t, "d", cfg.kv["C"].StringValue)
}

func TestNewFromMultipleFiles_OverridesCommonPropsAccordingToFileNameOrder(t *testing.T) {
	f1 := files.NewTempFileWithContent(t, `{"a":"b"}`)
	defer files.RemoveSilently(f1)
	f2 := files.NewTempFileWithContent(t, `{"a":"c"}`)
	defer files.RemoveSilently(f2)

	cfg, err := NewLoader(f1, f2).Load()
	require.NoError(t, err, "failed reading config files")

	require.Equal(t, "c", cfg.kv["A"].StringValue, "later config file did not override earlier config file's prop")
}

func TestNewFromMultipleFiles_ForE2E(t *testing.T) {
	cfg, err := NewLoader("../docker/test/benchmark-config/node1.json").Load()
	require.NoError(t, err, "failed parsing config file")

	require.EqualValues(t, "a328846cd5b4979d68a8c58a9bdfeee657b34de7", cfg.NodeAddress().String())
}

func TestConfigLoader_CallsReconfigureOnFileChange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	f1 := files.NewTempFileWithContent(t, `{"a":"b"}`)
	defer files.RemoveSilently(f1)
	f2 := files.NewTempFileWithContent(t, `{"c":"d"}`)
	defer files.RemoveSilently(f2)

	loader := NewLoader(f1, f2)

	ch := make(chan *MapBasedConfig)
	loader.OnConfigChanged(func(newConfig *MapBasedConfig) {
		ch <- newConfig
	})

	// load so that we initialize our state
	_, err := loader.Load()
	require.NoError(t, err, "failed initial loading of config file")

	loader.ListenForChanges(ctx, log.DefaultTestingLogger(t))

	require.NoError(t, ioutil.WriteFile(f1, []byte(`{"a":"b1"}`), 0644))
	select {
	case <-ctx.Done():
		t.Errorf("config change event not triggered")
	case cfg := <-ch:
		require.Equal(t, "b1", cfg.kv["A"].StringValue, "config change not reflected in new config")
	}

	require.NoError(t, ioutil.WriteFile(f2, []byte(`{"c":"d1"}`), 0644))
	select {
	case <-ctx.Done():
		t.Errorf("config change event not triggered")
	case cfg := <-ch:
		require.Equal(t, "b1", cfg.kv["A"].StringValue, "config change not reflected in new config")
		require.Equal(t, "d1", cfg.kv["C"].StringValue, "config change not reflected in new config")
	}
}
