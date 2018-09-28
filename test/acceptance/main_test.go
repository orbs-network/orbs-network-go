package acceptance

import (
	"github.com/orbs-network/orbs-network-go/config"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("NO_LOG_STDOUT") == "true" {
		logs := config.GetProjectSourceRootPath() + "/logs/acceptance/"
		os.RemoveAll(logs)
		os.MkdirAll(logs, 0755)
	}

	os.Exit(m.Run())
}
