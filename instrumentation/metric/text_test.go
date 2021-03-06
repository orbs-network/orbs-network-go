// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestText(t *testing.T) {
	r := NewRegistry()
	text := r.NewText("hello")
	require.EqualValues(t, "", text.Value().(string))

	text.Update("world")
	require.EqualValues(t, "world", text.Value().(string))
}

func TestTextWithDefaultValue(t *testing.T) {
	r := NewRegistry()
	text := r.NewText("hello", "new default")
	require.EqualValues(t, "new default", text.Value().(string))

	text.Update("world")
	require.EqualValues(t, "world", text.Value().(string))
}
