//+build !race

package e2e

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/processor/javascript/test"
	"path"
)

func buildDummyPlugin() {
	test.BuildDummyPlugin("services/processor/plugins/dummy/", "test/e2e/dummy_plugin.bin")
}

func dummyPluginPath() string {
	return path.Join(config.GetProjectSourceRootPath(), "test/e2e/dummy_plugin.bin")
}
