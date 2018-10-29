package synchronization

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"testing"
)

type report struct {
	message string
	fields []*log.Field
}

type collector struct {
	errors chan report

}

func (c *collector) Error(message string, fields ...*log.Field) {
	c.errors <- report{message, fields}
}

func mockLogger() *collector {
	c := &collector{errors: make(chan report)}
	return c
}

func TestRunSupervised_ReportsOnPanic(t *testing.T) {
	logger := mockLogger()

	require.NotPanicsf(t, func() {
		RunSupervised(logger, func() {
			panic("foo")
		})
	}, "RunSupervised panicked unexpectedly")

	report := <-logger.errors
	require.Equal(t, report.message, "recovered panic")
	require.Contains(t, report.fields[0].Value(), "foo")
}
