package adapter

import (
	"context"
	"encoding/hex"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"time"
)

const SOURCE_CODE_PATH = "native-src"
const SHARED_OBJECT_PATH = "native-bin"
const GC_CACHE_PATH = "native-cache"
const MAX_COMPILATION_TIME = 5 * time.Second // TODO: maybe move to config or maybe have caller provide via context

type Config interface {
	ProcessorArtifactPath() string
}

type nativeCompiler struct {
	config    Config
	reporting log.BasicLogger
}

func NewNativeCompiler(config Config, reporting log.BasicLogger) Compiler {
	reporting = reporting.For(log.String("adapter", "processor-native"))
	c := &nativeCompiler{
		config:    config,
		reporting: reporting,
	}

	// warm up compilation cache
	_, err := c.Compile(string(contracts.SourceCodeForNop()))
	if err != nil {
		reporting.Error("warm up compilation on init failed", log.Error(err))
	}

	return c
}

func (c *nativeCompiler) Compile(code string) (*sdk.ContractInfo, error) {
	artifactsPath := c.config.ProcessorArtifactPath()
	hashOfCode := getHashOfCode(code)

	sourceCodeFilePath, err := writeSourceCodeToDisk(hashOfCode, code, artifactsPath)
	defer os.Remove(sourceCodeFilePath)
	if err != nil {
		return nil, err
	}

	soFilePath, err := buildSharedObject(hashOfCode, sourceCodeFilePath, artifactsPath)
	if err != nil {
		return nil, err
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

func buildSharedObject(filenamePrefix string, sourceFilePath string, artifactsPath string) (string, error) {
	dir := filepath.Join(artifactsPath, SHARED_OBJECT_PATH)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}
	soFilePath := filepath.Join(dir, filenamePrefix) + ".so"

	// if the file is currently loaded as plugin, we won't be able to delete and it's ok
	if _, err := os.Stat(soFilePath); err == nil {
		err = os.Remove(soFilePath)
		if err != nil {
			return soFilePath, nil
		}
	}

	// compile
	ctx, cancel := context.WithTimeout(context.Background(), MAX_COMPILATION_TIME)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-buildmode=plugin", "-o", soFilePath, sourceFilePath)

	cmd.Env = []string{
		"GOPATH=" + getGOPATH(),
		"PATH=" + os.Getenv("PATH"),
		"GOCACHE=" + filepath.Join(artifactsPath, GC_CACHE_PATH),
		// "GOGC=off", (this improves compilation time by a small factor)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// "go build", invoked with a file name, puts this odd message before any compile errors; strip it.
			errs := strings.Replace(string(out), "# command-line-arguments\n", "", 1)
			errs = strings.Replace(errs, "\n", "; ", -1)
			return "", errors.Errorf("error building go source: %v", errs)
		}
		return "", errors.Errorf("error building go source: %v", err)
	}

	return soFilePath, nil
}

func loadSharedObject(soFilePath string) (*sdk.ContractInfo, error) {
	loadedPlugin, err := plugin.Open(soFilePath)
	if err != nil {
		return nil, err
	}

	contractSymbol, err := loadedPlugin.Lookup("CONTRACT")
	if err != nil {
		return nil, err
	}

	return contractSymbol.(*sdk.ContractInfo), nil
}

func getGOPATH() string {
	res := os.Getenv("GOPATH")
	if res == "" {
		return filepath.Join(os.Getenv("HOME"), "go")
	}
	return res
}
