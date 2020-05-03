package config

import (
	"fmt"
	"io/ioutil"
	"runtime/debug"
	"strings"
)

/**
	These functions are used to align & update the orbs-contract-sdk version found
	in our main go.mod file of orbs-network-go to the go.mod used for contract compilations
	being run in the e2e tests. This harness copies the template go.mod (same as happens during a CI docker build for our binary)
	to mimick the same behavior. If these functions are not used, the go.mod template would not contain a valid version of the SDK to import
	by the compiler and native contract deployments will fail to build.

	The template doesn't contain the same version as in the main go.mod file to keep things DRY and avoid mismatches
**/

type ArtifactsDependencyVersions struct {
	SDK_VER      string
	X_CRYPTO_VER string
}

func extractGoModVersionFromGoMod(input []byte, dependency string) (version string) {
	goModLines := strings.Split(string(input), "\n")
	for _, line := range goModLines {
		if strings.Contains(line, dependency) {
			parts := strings.Split(strings.Trim(line, "\t\n"), " ")
			version = parts[1]
		}
	}

	return
}

func extractGoModVersionFromDebug(info *debug.BuildInfo, dependency string) (version string) {
	for _, mod := range info.Deps {
		if mod.Path == dependency {
			version = mod.Version
		}
	}

	return
}

func GetMainProjectDependencyVersions(pathToMainGoMod string) ArtifactsDependencyVersions {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		input, err := ioutil.ReadFile(pathToMainGoMod)
		if err != nil {
			panic(fmt.Sprintf("failed to read file: %s", err.Error()))
		}

		return ArtifactsDependencyVersions{
			SDK_VER:      extractGoModVersionFromGoMod(input, "github.com/orbs-network/orbs-contract-sdk"),
			X_CRYPTO_VER: extractGoModVersionFromGoMod(input, "golang.org/x/crypto"),
		}
	}

	return ArtifactsDependencyVersions{
		SDK_VER:      extractGoModVersionFromDebug(info, "github.com/orbs-network/orbs-contract-sdk"),
		X_CRYPTO_VER: extractGoModVersionFromDebug(info, "golang.org/x/crypto"),
	}
}
