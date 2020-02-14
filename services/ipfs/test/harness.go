package test

import (
	"github.com/mr-tron/base58"
	"github.com/orbs-network/orbs-network-go/config"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var EXAMPLE_JSON_HASH, _ = base58.Decode("QmZCibsKZbJtwUgfvJUQeyKor7i5XqKimCVHp5qpUmS26d")

type harness struct {
	daemon *exec.Cmd
	env    []string
}

func NewIPFSDaemonHarness() *harness {
	ipfsPath := filepath.Join(config.GetProjectSourceTmpPath(), ".ipfs")
	os.RemoveAll(ipfsPath)

	env := []string{
		"HOME=" + config.GetProjectSourceTmpPath(),
	}

	init := exec.Command("ipfs", "init")
	init.Env = env
	//init.Stderr = os.Stderr
	//init.Stdout = os.Stdout
	init.Run()

	daemon := exec.Command("ipfs", "daemon")
	//daemon.Stderr = os.Stderr
	//daemon.Stdout = os.Stdout
	daemon.Env = env

	return &harness{
		daemon: daemon,
		env:    env,
	}
}

func (h *harness) StartDaemon() error {
	err := h.daemon.Start()
	<-time.After(3 * time.Second)
	return err
}

func (h *harness) StopDaemon() error {
	return h.daemon.Process.Kill()
}

func (h *harness) AddFile(path string) error {
	add := exec.Command("ipfs", "add", path)
	add.Env = h.env
	//add.Stdout = os.Stdout
	//add.Stderr = os.Stderr

	return add.Run()
}

func ExampleJSONPath() string {
	return filepath.Join(filepath.Join(config.GetProjectSourceRootPath(), "services", "ipfs", "test", "_data", "example.json"))
}
