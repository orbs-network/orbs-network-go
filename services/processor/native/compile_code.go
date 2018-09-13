package native

import (
	"context"
	"encoding/hex"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const sourceCodePath = "src"
const sharedObjectPath = "bin"
const maxCompilationTime = 3 * time.Second // TODO: maybe move to config

func compileAndLoadDeployedSourceCode(code string, artifactsPath string) (*sdk.ContractInfo, error) {
	hashOfCode := getHashOfCode(code)

	sourceCodeFilePath, err := writeSourceCodeToDisk(hashOfCode, code, artifactsPath)
	defer os.Remove(sourceCodeFilePath)
	if err != nil {
		return nil, err
	}

	_, err = buildSharedObject(hashOfCode, sourceCodeFilePath, artifactsPath)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func getHashOfCode(code string) string {
	return hex.EncodeToString(hash.CalcSha256([]byte(code)))
}

func writeSourceCodeToDisk(filenamePrefix string, code string, artifactsPath string) (string, error) {
	dir := filepath.Join(artifactsPath, sourceCodePath)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}
	sourceFilePath := filepath.Join(dir, filenamePrefix) + ".go"

	err = ioutil.WriteFile(sourceFilePath, []byte(code), 0644)
	if err != nil {
		return "", err
	}

	return sourceFilePath, nil
}

func buildSharedObject(filenamePrefix string, sourceFilePath string, artifactsPath string) (string, error) {
	dir := filepath.Join(artifactsPath, sharedObjectPath)
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
	ctx, cancel := context.WithTimeout(context.Background(), maxCompilationTime)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-buildmode=plugin", "-o", soFilePath, sourceFilePath)

	cmd.Env = []string{"GOPATH=" + os.Getenv("GOPATH")}

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
