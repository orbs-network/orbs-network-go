package acceptance

import (
	"github.com/orbs-network/orbs-network-go/config"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	logs := config.GetProjectSourceRootPath() + "/_logs/acceptance/"
	os.RemoveAll(logs)
	os.MkdirAll(logs, 0755)

	os.Exit(m.Run())
}
