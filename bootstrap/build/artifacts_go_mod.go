package main

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"os"
	"path/filepath"
)

func main() {
	versions := config.GetMainProjectDependencyVersions(config.GetProjectSourceRootPath())
	artifactsPath := os.Args[1]

	os.MkdirAll(artifactsPath, 0755)

	goModPath := filepath.Join(artifactsPath, "go.mod")
	if err := adapter.WriteArtifactsGoModToDisk(goModPath, versions); err != nil {
		panic(err)
	}
}
