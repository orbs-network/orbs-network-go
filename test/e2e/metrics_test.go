package e2e

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"
)

func TestE2E_Metrics(t *testing.T) {
	h := NewAppHarness()

	if h.config.RemoteEnvironment {
		t.Skip("Skipping E2E metrics is only for local testing")
	}
	h.WaitUntilTransactionPoolIsReady(t) // warm up

	// check reading status.json
	res, err := http.Get(h.config.AppChainUrl + "/status")
	require.NoError(t, err, "should succeed reading status endpoint")
	require.NotNil(t, res, "status should not be empty")

	readBytes, _ := ioutil.ReadAll(res.Body)
	localWrite(t, readBytes, "status.json")

	// check reading of metrics.json
	res, err = http.Get(h.metricsUrl)
	require.NoError(t, err, "should succeed reading metrics endpoint")
	require.NotNil(t, res, "metrics should not be empty")

	readBytes, _ = ioutil.ReadAll(res.Body)
	localWrite(t, readBytes, "metrics.json")

	// check reader
	metricReader, err := metric.NewReader(h.metricsUrl)
	require.NoError(t, err, "should succeed reading metrics back into map")
	require.NotEqual(t, 0, len(metricReader), "should not be empty")

	blockHeight, found := metricReader.GetAsInt(blockstorage.MetricBlockHeight)
	require.True(t, found, "block height should be found")
	require.True(t, blockHeight > int64(CannedBlocksFileMinHeight))
}

func localWrite(t *testing.T, data []byte, filename string) {
	// uncomment to save the values to a file for testing
	err := ioutil.WriteFile(filepath.Join(config.GetCurrentSourceFileDirPath(), "_data", filename), data, 0666)
	require.NoError(t, err, "should be able to w")
}
