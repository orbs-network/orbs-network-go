// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/stretchr/testify/require"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
)

// uses the go test runner "go test" to run a test with an identical name
// in the _supervised_in_test directory and takes expectations regarding output
func executeGoTestRunner(t *testing.T, expectedLogs []string, unexpectedLogs []string) {
	out, _ := exec.Command(
		path.Join(runtime.GOROOT(), "bin", "go"),
		"test",
		"github.com/orbs-network/orbs-network-go/synchronization/supervised/_supervised_in_test/",
		"-v",
		"-run",
		"^("+t.Name()+")$").CombinedOutput()

	output := string(out)

	// debug print output
	//fmt.Println("\n >>>>>>>>>>>>>>>>>>>>>>>>>>>> DEBUG PRINT\n", output, "<<<<<<<<<<<<<<<<<<<<<<<<<<<< DEBUG PRINT")

	for _, logLine := range expectedLogs {
		require.Truef(t, strings.Contains(output, logLine), "log should contain: '%s'", logLine)
	}
	for _, logLine := range unexpectedLogs {
		require.Falsef(t, strings.Contains(output, logLine), "log should not contain: '%s'", logLine)
	}
}
