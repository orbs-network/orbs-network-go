// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"encoding/json"
	"sync/atomic"
)

type Text struct {
	name  string
	pName string
	value atomic.Value
}

func newText(name string, pName string, defaultValue ...string) *Text {
	value := ""

	if len(defaultValue) == 1 {
		value = defaultValue[0]
	}

	res := &Text{
		name:  name,
		pName: prometheusName(pName),
		value: atomic.Value{},
	}
	res.value.Store(value)
	return res
}

func (t *Text) Name() string {
	return t.name
}

func (t *Text) Export() interface{} {
	value := t.value.Load().(string)
	var x []interface{}
	if err := json.Unmarshal([]byte(value), &x); err == nil {
		return x
	} else {
		return value
	}
}

func (t *Text) Update(value string) {
	t.value.Store(value)
}

func (t *Text) Value() interface{} {
	return t.value.Load().(string)
}
