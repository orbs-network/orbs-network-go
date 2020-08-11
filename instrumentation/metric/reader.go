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
	"strings"
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

	status := struct {
		Payload   exportedMap
	}{}
	err = json.Unmarshal(readBytes, &status)
	if err != nil {
		return nil, err
	}
	return status.Payload, nil
}

func (mr exportedMap) GetAsInt(name string) (int64, bool) {
	value, found := mr.Get(name)
	if found {
		if valueAsInt, ok := value.(int64); ok {
			return valueAsInt, true
		} else if valueAsFloat, ok := value.(float64); ok {
			return int64(valueAsFloat), true
		}
	}
	return 0, false
}

func (mr exportedMap) Get(name string) (interface{}, bool) {
	nameParts := deconstructNesting(name)
	return travelNesting(mr, nameParts)
}

func deconstructNesting(name string) []string {
	nameParts := strings.Split(name, ".")
	nameLen := len(nameParts)

	knownType := isNameAKnownType(nameParts[nameLen-1])
	if nameLen > 1 && knownType {
		nameParts = nameParts[:nameLen-1]
	}
	return nameParts
}

// NOTE: due to the origin of the maps here from unmarshal, GO doesn't type assert to exportedMap even thought it is the same so i explicitly use the full type
func travelNesting(currLevel map[string]interface {}, nameParts []string) (interface{}, bool) {
	nameLen := len(nameParts)
	for i := 0; i < nameLen; i++ {
		if value, exists := currLevel[nameParts[i]]; !exists { // is in current level map ?
			return nil, false
		} else if valueAsMap, ok := value.(map[string]interface {}); !ok { // is this level a map or a leaf
			if i == nameLen-1 {
				return value, true // exact match
			} else {
				return nil, false // there are more names but we are at end not match
			}
		} else {
			currLevel = valueAsMap // go one level in
		}
	}

	if value, exists := currLevel["Value"]; exists {
		return value, true // special case where end of name is both branch & leaf
	}
	return nil, false
}
