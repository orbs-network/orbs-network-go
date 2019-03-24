// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

//+build !nonativecompiler

package adapter

import (
	"context"
	"encoding/hex"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"
)

var LogTag = log.String("adapter", "processor-native")

type nativeCompiler struct {
	config Config
	logger log.BasicLogger
}

func NewNativeCompiler(config Config, logger log.BasicLogger) Compiler {
	c := &nativeCompiler{
		config: config,
		logger: logger.WithTags(LogTag),
	}

	c.warmUpCompilationCache() // so next compilations take 200 ms instead of 2 sec

	return c
}

func (c *nativeCompiler) warmUpCompilationCache() {
	ctx, cancel := context.WithTimeout(context.Background(), MAX_WARM_UP_COMPILATION_TIME)
	defer cancel()

	_, err := c.Compile(ctx, string(contracts.SourceCodeForNop()))
	if err != nil {
		c.logger.Error("warm up compilation on init failed", log.Error(err))
	}
}

func (c *nativeCompiler) Compile(ctx context.Context, code string) (*sdkContext.ContractInfo, error) {
	artifactsPath := c.config.ProcessorArtifactPath()
	hashOfCode := getHashOfCode(code)

	sourceCodeFilePath, err := writeSourceCodeToDisk(hashOfCode, code, artifactsPath)
	defer os.Remove(sourceCodeFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "could not write source code to disk")
	}

	soFilePath, err := buildSharedObject(ctx, hashOfCode, sourceCodeFilePath, artifactsPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not build a shared object")
	}

	return loadSharedObject(soFilePath)
}

func getHashOfCode(code string) string {
	return hex.EncodeToString(hash.CalcSha256([]byte(code)))
}

func writeSourceCodeToDisk(filenamePrefix string, code string, artifactsPath string) (string, error) {
	dir := filepath.Join(artifactsPath, SOURCE_CODE_PATH)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}
	sourceFilePath := filepath.Join(dir, filenamePrefix) + ".go"

	err = ioutil.WriteFile(sourceFilePath, []byte(code), 0600)
	if err != nil {
		return "", err
	}

	return sourceFilePath, nil
}

func buildSharedObject(ctx context.Context, filenamePrefix string, sourceFilePath string, artifactsPath string) (string, error) {
	dir := filepath.Join(artifactsPath, SHARED_OBJECT_PATH)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}
	soFilePath := filepath.Join(dir, filenamePrefix) + ".so"

	// if the file is currently loaded as plugin, we won't be able to delete and it's ok
	if _, err = os.Stat(soFilePath); err == nil {
		err = os.Remove(soFilePath)
		if err != nil {
			return soFilePath, nil
		}
	}

	// compile
	goCmd := path.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.CommandContext(ctx, goCmd, "build", "-buildmode=plugin", "-o", soFilePath, sourceFilePath)
	cmd.Env = []string{
		"GOPATH=" + getGOPATH(),
		"PATH=" + os.Getenv("PATH"),
		"GOCACHE=" + filepath.Join(artifactsPath, GC_CACHE_PATH),
		// "GOGC=off", (this improves compilation time by a small factor)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		buildOutput := string(out)
		buildOutput = strings.Replace(buildOutput, "# command-line-arguments\n", "", 1) // "go build", invoked with a file name, puts this odd message before any compile errors; strip it.
		buildOutput = strings.Replace(buildOutput, "\n", "; ", -1)
		return "", errors.Errorf("error building go source: %s, go build output: %s", err.Error(), buildOutput)
	}

	return soFilePath, nil
}

func loadSharedObject(soFilePath string) (*sdkContext.ContractInfo, error) {
	loadedPlugin, err := plugin.Open(soFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "could not open plugin")
	}

	publicMethods := []interface{}{}
	var publicMethodsPtr *[]interface{}
	publicMethodsSymbol, err := loadedPlugin.Lookup("PUBLIC")
	if err != nil {
		return nil, errors.Wrap(err, "could not look up a symbol inside a plugin")
	}
	publicMethodsPtr, ok := publicMethodsSymbol.(*[]interface{})
	if !ok {
		return nil, errors.New("PUBLIC methods export has incorrect type")
	}
	publicMethods = *publicMethodsPtr

	systemMethods := []interface{}{}
	var systemMethodsPtr *[]interface{}
	systemMethodsSymbol, err := loadedPlugin.Lookup("SYSTEM")
	if err == nil {
		systemMethodsPtr, ok = systemMethodsSymbol.(*[]interface{})
		if !ok {
			return nil, errors.New("SYSTEM methods export has incorrect type")
		}
		systemMethods = *systemMethodsPtr
	}

	eventsMethods := []interface{}{}
	var eventsMethodsPtr *[]interface{}
	eventsMethodsSymbol, err := loadedPlugin.Lookup("EVENTS")
	if err == nil {
		eventsMethodsPtr, ok = eventsMethodsSymbol.(*[]interface{})
		if !ok {
			return nil, errors.New("EVENTS methods export has incorrect type")
		}
		eventsMethods = *eventsMethodsPtr
	}

	return &sdkContext.ContractInfo{
		PublicMethods: publicMethods,
		SystemMethods: systemMethods,
		EventsMethods: eventsMethods,
		Permission:    sdkContext.PERMISSION_SCOPE_SERVICE, // we don't support compiling system contracts on the fly
	}, nil
}

func getGOPATH() string {
	res := os.Getenv("GOPATH")
	if res == "" {
		return filepath.Join(os.Getenv("HOME"), "go")
	}
	return res
}
