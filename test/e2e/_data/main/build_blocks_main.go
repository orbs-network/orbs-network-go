package main

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/e2e"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func main() {

	fmt.Printf("where am i? %s", config.GetCurrentSourceFileDirPath())

	filename := filepath.Join(config.GetCurrentSourceFileDirPath(), "..", "blocks")

	err := os.Truncate(filename, 0)
	if err != nil {
		fmt.Printf("error cleaning file, %s: %s", filename, err)
		os.Exit(1)
	}

	n := e2e.NewInProcessE2ENetwork()
	time.Sleep(time.Minute)
	n.GracefulShutdown()

	nodeFolders, err := ioutil.ReadDir(e2e.BlockStorageDataDirPrefix)
	if err != nil {
		fmt.Printf("error searching for e2e blocks file floders: %s", err)
		os.Exit(1)
	}

	copyFile(filepath.Join(e2e.BlockStorageDataDirPrefix, nodeFolders[0].Name(), "blocks"), filename)
}

func copyFile(sourcePath string, targetPath string) {
	rawBlocks, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		panic("failed loading blocks file")
	}
	err = ioutil.WriteFile(targetPath, rawBlocks, 0644)
	if err != nil {
		panic(fmt.Sprintf("failed copying files: %s -> %s", sourcePath, targetPath))
	}
}
