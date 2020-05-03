package adapter

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/pkg/errors"
	"io/ioutil"
	"text/template"
)

const GO_MOD_TEMPLATE = `
module processor_native_src

go 1.12

// This go.mod is inserted into the Docker images we deliver for testnet/mainnet usage
// To instruct the native compiler to use the correct versions for these go modules
// so that built contracts won't break the system.

require (
	github.com/orbs-network/orbs-contract-sdk {{.SDK_VER}}
	golang.org/x/crypto {{.X_CRYPTO_VER}}
)
`

func writeArtifactsGoModToDisk(targetFilePath string, versions config.ArtifactsDependencyVersions) error {
	t, err := template.New("go.mod.template").Parse(GO_MOD_TEMPLATE)
	if err != nil {
		return errors.Wrap(err, "failed to parse go.mod.template file")
	}

	output := bytes.NewBufferString("")
	if err := t.Execute(output, versions); err != nil {
		return errors.Wrap(err, "failed to execute go.mod.template file")
	}

	if err = ioutil.WriteFile(targetFilePath, output.Bytes(), 0666); err != nil {
		return errors.Wrap(err, "failed to re-write e2e go.mod file")
	}

	return nil
}
