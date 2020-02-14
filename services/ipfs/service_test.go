package ipfs

import (
	"context"
	ipfsClient "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestWithLocalNode(t *testing.T) {
	api, err := ipfsClient.NewLocalApi()
	if err != nil {
		panic(err)
	}

	r, err := api.Object().Data(context.Background(), path.New("QmUAWLL8kx7FDhsgiMC8nCP1xcuqkCh6mhDZzqvA3U3fUF"))
	if err != nil {
		panic(err)
	}

	contents, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	readme, err := ioutil.ReadFile(filepath.Join(config.GetProjectSourceRootPath(), "README.md"))
	rawContents := contents[5:len(contents)-3]
	require.EqualValues(t, string(readme), string(rawContents))
}
