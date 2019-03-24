// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
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

func GetNodesDataDirs() ([]string, error) {
	nodeFolders, err := ioutil.ReadDir(blockStorageDataDirPrefix)
	if err != nil {
		return nil, err
	}

	var nodeDataDirs []string
	for _, nodeFolder := range nodeFolders {
		nodeDataDirs = append(nodeDataDirs, filepath.Join(blockStorageDataDirPrefix, nodeFolder.Name(), "blocks"))
	}

	return nodeDataDirs, nil
}

func getProcessorArtifactPath() (string, string) {
	dir := filepath.Join(config.GetCurrentSourceFileDirPath(), "_tmp")
	return filepath.Join(dir, "processor-artifacts"), dir
}

func cleanNativeProcessorCache() {
	_, dirToCleanup := getProcessorArtifactPath()
	_ = os.RemoveAll(dirToCleanup)
}

func cleanBlockStorage() {
	_ = os.RemoveAll(blockStorageDataDirPrefix)
}

func deployBlockStorageFiles(targetDir string, logger log.BasicLogger) {
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
