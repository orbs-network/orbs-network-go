// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

func NewReader(endpoint string) (exportedMap, error) {
	res, err := http.Get(endpoint)
	if err != nil {
		return nil, errors.Errorf("MetricReader cannot read data from endpoint '%s'", endpoint)
	}
	if res == nil {
		return nil, err
	}

	readBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	m := make(exportedMap)
	err = json.Unmarshal(readBytes, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (mr exportedMap) Get(name string) (interface{}, bool) {
	if value, exists := mr[name]; exists {
		return value, true
	}
	// TODO try read nested
	return nil, false
}

func (mr exportedMap) GetAsInt(name string) (int64, bool) {
	value, found := mr.Get(name)
	if found {
		if valueAsFloat, ok := value.(float64); ok {
			return int64(valueAsFloat), true
		}
	}
	return 0, false
}
