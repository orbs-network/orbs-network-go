package commands

import (
	"bytes"
	"flag"
	"fmt"
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

func HandleStartCommand(args []string) {
	flagSet := flag.NewFlagSet("start", flag.ExitOnError)

	binaryPtr := flagSet.String("binaryPath", "", "Provide your own path to a pre-compiled gamma binary")
	portPtr := flagSet.String("port", "8080", "The port to bind the gamma-server on")

	flagSet.Parse(args)

	var lookups []string

	if *binaryPtr != "" {
		lookups = append(lookups, *binaryPtr)
	}

	lookups = append(lookups, "/usr/local/bin/gamma-server", "gamma-server")

	for _, binaryPath := range lookups {
		_, err := os.Stat(binaryPath)
		if err == nil {
			// Found a workable binary , let's execute it.
			fmt.Println(fmt.Sprintf("gamma-server started and listening on port %s", *portPtr))

			execCommand := []string{binaryPath, "-port", *portPtr, "&>/dev/null", "&"}
			execCommand[0] = "./" + execCommand[0]
			runCommand(execCommand)
			os.Exit(0)
		}
	}

	fmt.Println("Could not find gamma-server on this machine")
	os.Exit(1)
}
