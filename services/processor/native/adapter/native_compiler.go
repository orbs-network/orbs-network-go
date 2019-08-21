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
	"fmt"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"
	"time"
)

var LogTag = log.String("adapter", "processor-native")

type nativeCompilerMetrics struct {
	lastWarmUpTimeMs *metric.Gauge
	totalCompileTime *metric.Histogram
	writeToDiskTime  *metric.Histogram
	buildTime        *metric.Histogram
	loadTime         *metric.Histogram
	sourceSize       *metric.Histogram
}

type nativeCompiler struct {
	config  Config
	logger  log.Logger
	metrics *nativeCompilerMetrics
}

func createNativeCompilerMetrics(factory metric.Factory) *nativeCompilerMetrics {
	return &nativeCompilerMetrics{
		buildTime:        factory.NewLatency("Processor.Native.Compiler.Build.Time.Millis", 60*time.Minute),
		totalCompileTime: factory.NewLatency("Processor.Native.Compiler.Total.Compile.Time.Millis", 60*time.Minute),
		loadTime:         factory.NewLatency("Processor.Native.Compiler.LoadObject.Time.Millis", 60*time.Minute),
		lastWarmUpTimeMs: factory.NewGauge("Processor.Native.Compiler.LastWarmUp.Time.Millis"),
		writeToDiskTime:  factory.NewLatency("Processor.Native.Compiler.WriteToDisk.Time.Millis", 60*time.Minute),
		sourceSize:       factory.NewHistogram("Processor.Native.Compiler.Source.Size.Bytes", 1024*1024), // megabyte
	}
}

func NewNativeCompiler(config Config, parent log.Logger, factory metric.Factory) Compiler {
	logger := parent.WithTags(LogTag)
	c := &nativeCompiler{
		config:  config,
		logger:  logger,
		metrics: createNativeCompilerMetrics(factory),
	}

	if config.ProcessorPerformWarmUpCompilation() {
		c.warmUpCompilationCache() // so next compilations take 200 ms instead of 2 sec
	} else {
		logger.Info("skipping warm-up compilation")
	}

	return c
}

func (c *nativeCompiler) warmUpCompilationCache() {
	ctx, cancel := context.WithTimeout(context.Background(), MAX_WARM_UP_COMPILATION_TIME)
	defer cancel()

	start := time.Now()
	_, err := c.Compile(ctx, string(contracts.SourceCodeForNop()))
	c.metrics.lastWarmUpTimeMs.Update(time.Since(start).Nanoseconds() / int64(time.Millisecond))

	if err != nil {
		c.logger.Error("warm up compilation on init failed", log.Error(err))
	}
}

func (c *nativeCompiler) Compile(ctx context.Context, code ...string) (*sdkContext.ContractInfo, error) {
	c.metrics.sourceSize.Record(int64(len(code)))
	start := time.Now()
	defer c.metrics.totalCompileTime.RecordSince(start)

	artifactsPath := c.config.ProcessorArtifactPath()
	hashOfCode := getHashOfCode(code)

	writeTime := time.Now()
	sourceCodeFilePaths, err := writeSourceCodeToDisk(hashOfCode, code, artifactsPath)
	c.metrics.writeToDiskTime.RecordSince(writeTime)
	for _, sourceCodeFilePath := range sourceCodeFilePaths {
		defer os.Remove(sourceCodeFilePath)
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not write source code to disk")
	}

	buildTime := time.Now()
	soFilePath, err := buildSharedObject(ctx, hashOfCode, sourceCodeFilePaths, artifactsPath)
	c.metrics.buildTime.RecordSince(buildTime)
	if err != nil {
		return nil, errors.Wrap(err, "could not build a shared object")
	}

	loadSoTime := time.Now()
	so, err := loadSharedObject(soFilePath)
	c.metrics.loadTime.RecordSince(loadSoTime)

	return so, err
}

func getHashOfCode(code []string) string {
	var buffer string
	for _, c := range code {
		buffer += c
	}

	return hex.EncodeToString(hash.CalcSha256([]byte(buffer)))
}

func writeSourceCodeToDisk(filenamePrefix string, code []string, artifactsPath string) ([]string, error) {
	dir := filepath.Join(artifactsPath, SOURCE_CODE_PATH, filenamePrefix)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return nil, err
	}

	var sourceFilePaths []string
	for i, c := range code {
		sourceFilePath := filepath.Join(dir, fmt.Sprintf("contract.%d.go", i))
		sourceFilePaths = append(sourceFilePaths, sourceFilePath)

		err = ioutil.WriteFile(sourceFilePath, []byte(c), 0600)
		if err != nil {
			return nil, err
		}
	}

	return sourceFilePaths, nil
}

func buildSharedObject(ctx context.Context, filenamePrefix string, sourceFilePaths []string, artifactsPath string) (string, error) {
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
	cmdArgs := []string{"build", "-buildmode=plugin", "-o", soFilePath}
	cmdArgs = append(cmdArgs, sourceFilePaths...)
	cmd := exec.CommandContext(ctx, goCmd, cmdArgs...)
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
