package commands

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"os/exec"
)

func runCommand(command []string) {
	cmd := exec.Command(command[0], command[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		fmt.Println("Could not start gamma-server", err)
		os.Exit(1)
	}
}

func findGammaServerBinary(pathToBinary string) string {
	var lookups []string

	if pathToBinary != "" {
		lookups = append(lookups, pathToBinary)
	}

	lookups = append(lookups, "./gamma-server", "/usr/local/bin/gamma-server")

	for _, binaryPath := range lookups {
		_, err := os.Stat(binaryPath)
		if err == nil {
			return binaryPath
		}
	}

	return ""
}

func (r *CommandRunner) HandleStartCommand(args []string) (string, error) {

	flagSet := flag.NewFlagSet("start", flag.ExitOnError)

	binaryPtr := flagSet.String("binaryPath", "", "Provide your own path to a pre-compiled gamma binary")
	portPtr := flagSet.String("port", "8080", "The port to bind the gamma-server on")

	err := flagSet.Parse(args)
	if err != nil {
		return "", errors.Wrapf(err , "flag issues")
	}

	pathToBinary := findGammaServerBinary(*binaryPtr)
	if pathToBinary != "" {
		startMessage := `
gamma-server started and listening on port ` + *portPtr + `
For debugging/logging please run gamma-server directly

`
		execCommand := []string{pathToBinary, "-port", *portPtr, "&>/dev/null", "&"}
		runCommand(execCommand)

		return startMessage, nil
	}

	return "", errors.Errorf("Could not find gamma-server on this machine")
}
