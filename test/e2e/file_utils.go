// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"io/ioutil"
	"os"
	"path/filepath"
)

const blockStorageDataDirPrefix = "/tmp/orbs/e2e"

func CopyFile(sourcePath string, targetPath string) error {
	rawBlocks, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(targetPath, rawBlocks, 0644)
	if err != nil {
		return err
	}
	return nil
}

func GetNodesDataDirs(virtualChainId primitives.VirtualChainId) ([]string, error) {
	nodeFolders, err := ioutil.ReadDir(getVirtualChainDataDir(virtualChainId))
	if err != nil {
		return nil, err
	}

	var nodeDataDirs []string
	for _, nodeFolder := range nodeFolders {
		nodeDataDirs = append(nodeDataDirs, filepath.Join(getVirtualChainDataDir(virtualChainId), nodeFolder.Name(), "blocks"))
	}

	return nodeDataDirs, nil
}

func getVirtualChainDataDir(virtualChainId primitives.VirtualChainId) string {
	return filepath.Join(blockStorageDataDirPrefix, vChainPathComponent(virtualChainId))
}

func getProcessorArtifactPath(virtualChainId primitives.VirtualChainId) (string, string) {
	dir := filepath.Join(config.GetCurrentSourceFileDirPath(), "_tmp", vChainPathComponent(virtualChainId))
	return filepath.Join(dir, "processor-artifacts"), dir
}

func cleanNativeProcessorCache(virtualChainId primitives.VirtualChainId) {
	_, dirToCleanup := getProcessorArtifactPath(virtualChainId)
	_ = os.RemoveAll(dirToCleanup)
}

func cleanBlockStorage(virtualChainId primitives.VirtualChainId) {
	_ = os.RemoveAll(getVirtualChainDataDir(virtualChainId))
}

func deployBlockStorageFiles(targetDir string, logger log.Logger) {
	err := os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("could not create directory %s: %e", targetDir, err))
	}
	sourceBlocksFilePath := filepath.Join(config.GetCurrentSourceFileDirPath(), "_data", "blocks")
	targetBlocksFilePath := filepath.Join(targetDir, "blocks")

	logger.Info("copying blocks file", log.String("source", sourceBlocksFilePath), log.String("target", targetBlocksFilePath))

	err = CopyFile(sourceBlocksFilePath, targetBlocksFilePath)
	if err != nil {
		panic(fmt.Sprintf("could not copy files %s -> %s", sourceBlocksFilePath, targetBlocksFilePath))
	}
}

func vChainPathComponent(virtualChainId primitives.VirtualChainId) string {
	return fmt.Sprintf("vcid_%d", virtualChainId)
}
