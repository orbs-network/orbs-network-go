package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestText(t *testing.T) {
	r := NewRegistry()
	text := r.NewText("hello")
	require.EqualValues(t, "", text.Value())

	text.Update("world")
	require.EqualValues(t, "world", text.Value())
}

func TestTextWithDefaultValue(t *testing.T) {
	r := NewRegistry()
	text := r.NewText("hello", "new default")
	require.EqualValues(t, "new default", text.Value())

	text.Update("world")
	require.EqualValues(t, "world", text.Value())
}
