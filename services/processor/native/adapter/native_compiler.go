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
const MAX_COMPILATION_TIME = 5 * time.Second          // TODO: maybe move to config or maybe have caller provide via context
const MAX_WARM_UP_COMPILATION_TIME = 15 * time.Second // TODO: maybe move to config or maybe have caller provide via context

var LogTag = log.String("adapter", "processor-native")

type Config interface {
	ProcessorArtifactPath() string
}

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

func (c *nativeCompiler) Compile(ctx context.Context, code string) (*sdk.ContractInfo, error) {
	artifactsPath := c.config.ProcessorArtifactPath()
	hashOfCode := getHashOfCode(code)

	sourceCodeFilePath, err := writeSourceCodeToDisk(hashOfCode, code, artifactsPath)
	defer os.Remove(sourceCodeFilePath)
	if err != nil {
		return nil, err
	}

	soFilePath, err := buildSharedObject(ctx, hashOfCode, sourceCodeFilePath, artifactsPath)
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

func buildSharedObject(ctx context.Context, filenamePrefix string, sourceFilePath string, artifactsPath string) (string, error) {
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
	cmd := exec.CommandContext(ctx, "go", "build", "-buildmode=plugin", "-o", soFilePath, sourceFilePath)
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
