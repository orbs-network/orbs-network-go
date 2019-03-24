// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package main

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/e2e"
	"os"
	"path/filepath"
	"time"
)

// utility to generate e2e compatible blocks file.
// runs a fresh e2e network with no preexisting blocks for 1 minute, allowing it to close blocks
// after one minute shuts down the in memory network and extracts the blocks file to ../blocks

// NOTE overrides the same file used in e2e tests. on failure, any changes to blocks file must be reverted
func main() {

	clearBlocksFile()

	n := e2e.NewInProcessE2ENetwork()

	time.Sleep(time.Minute) // accumulate blocks

	extractBlocksFile()

	n.GracefulShutdownAndWipeDisk()
}

func cannedBlocksFilename() string {
	return filepath.Join(config.GetCurrentSourceFileDirPath(), "..", "blocks")
}

func clearBlocksFile() {
	err := os.Truncate(cannedBlocksFilename(), 0)
	if err != nil {
		fmt.Printf("error cleaning file, %s: %s", cannedBlocksFilename(), err)
		os.Exit(1)
	}
}

func extractBlocksFile() {
	nodeFolders, err := e2e.GetNodesDataDirs()
	if err != nil {
		fmt.Printf("error searching for e2e blocks file floders: %s", err)
		os.Exit(1)
	}
	err = e2e.CopyFile(nodeFolders[0], cannedBlocksFilename())
	if err != nil {
		fmt.Printf("could not copy files %s -> %s", nodeFolders[0], cannedBlocksFilename())
		os.Exit(1)
	}
}
