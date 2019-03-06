package test

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"os"
	"time"
)

type ethereumConnectorConfigForTests struct {
	endpoint                string
	privateKeyHex           string
	finalityTimeComponent   time.Duration
	finalityBlocksComponent uint32
}

func (c *ethereumConnectorConfigForTests) EthereumEndpoint() string {
	return c.endpoint
}

func (c *ethereumConnectorConfigForTests) EthereumFinalityTimeComponent() time.Duration {
	return c.finalityTimeComponent
}

func (c *ethereumConnectorConfigForTests) EthereumFinalityBlocksComponent() uint32 {
	return c.finalityBlocksComponent
}

func (c *ethereumConnectorConfigForTests) GetAuthFromConfig() (*bind.TransactOpts, error) {
	key, err := crypto.HexToECDSA(c.privateKeyHex)
	if err != nil {
		return nil, err
	}

	return bind.NewKeyedTransactor(key), nil
}

func ConfigForSimulatorConnection() *ethereumConnectorConfigForTests {
	return &ethereumConnectorConfigForTests{
		finalityTimeComponent:   0 * time.Millisecond,
		finalityBlocksComponent: 0,
	}
}

func ConfigForExternalRPCConnection() *ethereumConnectorConfigForTests {
	var cfg ethereumConnectorConfigForTests

	if endpoint := os.Getenv("ETHEREUM_ENDPOINT"); endpoint != "" {
		cfg.endpoint = endpoint
	}

	if privateKey := os.Getenv("ETHEREUM_PRIVATE_KEY"); privateKey != "" {
		cfg.privateKeyHex = privateKey
	}

	cfg.finalityTimeComponent = 10 * time.Second
	cfg.finalityBlocksComponent = 1

	return &cfg
}

func runningWithDocker() bool {
	return os.Getenv("EXTERNAL_TEST") == "true"
}
