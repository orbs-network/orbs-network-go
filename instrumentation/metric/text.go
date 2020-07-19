// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"sync/atomic"
)

type Text struct {
	name  string
	value atomic.Value
}

type textExport struct {
	Name  string
	Value string
}

func newText(name string, defaultValue ...string) *Text {
	value := ""

	if len(defaultValue) == 1 {
		value = defaultValue[0]
	}

	res := &Text{
		name:  name,
		value: atomic.Value{},
	}
	res.value.Store(value)
	return res
}

func (t *Text) Name() string {
	return t.name
}

func (t *Text) Export() exportedMetric {
	return textExport{
		t.name,
		t.value.Load().(string),
	}
}

func (t *Text) Update(value string) {
	t.value.Store(value)
}

func (t *Text) Value() interface{} {
	return t.value.Load().(string)
}
