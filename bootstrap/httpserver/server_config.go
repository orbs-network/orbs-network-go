package httpserver

import "github.com/orbs-network/orbs-network-go/config"

type ServerConfig struct {
	httpAddress string
	profiling   bool
}

func NewServerConfig(httpAddress string, profiling bool) config.HttpServerConfig {
	return &ServerConfig{
		httpAddress: httpAddress,
		profiling:   profiling,
	}
}

func (c *ServerConfig) HttpAddress() string {
	return c.httpAddress
}

func (c *ServerConfig) Profiling() bool {
	return c.profiling
}
