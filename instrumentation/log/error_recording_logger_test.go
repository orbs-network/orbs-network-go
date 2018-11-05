package log

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestErrorRecordingLogger_WithTags_PropagatesToNested(t *testing.T) {

	outer := NewErrorRecordingLogger(GetLogger(), nil)
	foobar := String("foo", "bar")
	inner := outer.WithTags(foobar)

	require.Contains(t, inner.Tags(), foobar)
}

