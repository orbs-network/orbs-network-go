package gamma

import (
	"flag"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func RunMain(t testing.TB, port int, overrideConfig string) int {
	require.NoError(t, flag.Set("override-config", overrideConfig))
	require.NoError(t, flag.Set("port", strconv.Itoa(port)))

	listeningPort := make(chan int)
	go Main(listeningPort)
	return <-listeningPort
}
