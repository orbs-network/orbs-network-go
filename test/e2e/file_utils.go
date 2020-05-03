// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const blockStorageDataDirPrefix = "/tmp/orbs/e2e"
const CannedBlocksFileMinHeight = 500

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
	dir := filepath.Join(os.TempDir(), "orbs", "processorArtifacts", vChainPathComponent(virtualChainId))
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

/**
	These functions are used to align & update the orbs-contract-sdk version found
	in our main go.mod file of orbs-network-go to the go.mod used for contract compilations
	being run in the e2e tests. This harness copies the template go.mod (same as happens during a CI docker build for our binary)
	to mimick the same behavior. If these functions are not used, the go.mod template would not contain a valid version of the SDK to import
	by the compiler and native contract deployments will fail to build.

	The template doesn't contain the same version as in the main go.mod file to keep things DRY and avoid mismatches
**/
func getMainProjectSDKVersion(pathToMainGoMod string) string {
	sdkVersion := ""

	input, err := ioutil.ReadFile(pathToMainGoMod)
	if err != nil {
		panic(fmt.Sprintf("failed to read file: %s", err.Error()))
	}

	goModLines := strings.Split(string(input), "\n")
	for _, line := range goModLines {
		if strings.Contains(line, "orbs-contract-sdk") {
			sdkParts := strings.Split(strings.Trim(line, "\t\n"), " ")
			sdkVersion = sdkParts[1]
		}
	}

	return sdkVersion
}

func replaceSDKVersion(targetFilePath string, sdkVersion string) {
	input, err := ioutil.ReadFile(targetFilePath)

	if err != nil {
		panic(fmt.Sprintf("failed to open e2e go.mod file for reading: %s", err.Error()))
	}

	output := bytes.Replace(input, []byte("SDK_VER"), []byte(sdkVersion), -1)

	if err = ioutil.WriteFile(targetFilePath, output, 0666); err != nil {
		panic(fmt.Sprintf("failed to re-write e2e go.mod file: %s", err.Error()))
	}
}

func setUpProcessorArtifactPath(virtualChainId primitives.VirtualChainId) string {
	processorArtifactPath, _ := getProcessorArtifactPath(virtualChainId)

	// copy go.mod file:
	err := os.MkdirAll(processorArtifactPath, 0755)
	if err != nil {
		panic(fmt.Sprintf("failed to make dir: %s", err.Error()))
	}

	mainGoModPath := filepath.Join(config.GetCurrentSourceFileDirPath(), "..", "..", "go.mod")
	sdkVersion := getMainProjectSDKVersion(mainGoModPath)

	goModTemplateFileName := "go.mod.template"

	sourceGoModPath := filepath.Join(config.GetCurrentSourceFileDirPath(), "..", "..", "docker/build", goModTemplateFileName)
	targetGoModPath := filepath.Join(processorArtifactPath, "go.mod")
	err = CopyFile(sourceGoModPath, targetGoModPath)
	if err != nil {
		panic(fmt.Sprintf("failed to copy go.mod file: %s", err.Error()))
	}

	fmt.Println("the target go.mod is at:", targetGoModPath, sdkVersion)
	replaceSDKVersion(targetGoModPath, sdkVersion)

	return processorArtifactPath
}
