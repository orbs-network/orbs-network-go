package metric

import "github.com/orbs-network/orbs-network-go/config"

func RegisterConfigIndicators(metricRegistry Registry, nodeConfig config.NodeConfig) {
	version := config.GetVersion()

	metricRegistry.NewText("Version.Semantic", version.Semantic)
	metricRegistry.NewText("Version.Commit", version.Commit)
	metricRegistry.NewText("Node.Address", nodeConfig.NodeAddress().String())
}
