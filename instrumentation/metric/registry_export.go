// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"strings"
)

func (r *inMemoryRegistry) ExportAllFlat() exportedMap {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make(exportedMap)
	for _, m := range r.mu.metrics {
		all[m.Name()] = m.Export()
	}

	return all
}

/*
 * Assumptions:
 * 1) Nested names seperated by '.'
 * 2) At least one '.' exists (must have one level of nesting)
 * 3) If last level is one of known types (see func isNameAKnownType) they will be "smartly typed"
 * 4) Cannot have in same level a scalar and next level: X.Y.Value and X.Y.Z.Value will be turned to x{y{value:Value, z {value}}}
 * 5) There is a e2e test that tries to call this function with panic if fail, this will prevent bad names
 */

func (r *inMemoryRegistry) ExportAllNested() exportedMap {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(exportedMap)

	for _, metric := range r.mu.metrics {
		metricValue := metric.Export()
		nameParts := strings.Split(metric.Name(), ".")

		parseNextNestedLevel(0, nameParts, result, metricValue)
	}

	return result
}

func parseNextNestedLevel(index int, nestedNameParts []string, currentNestedLevel exportedMap, value interface{}) {
	if index == len(nestedNameParts) - 1 {
		getOrCreateLeaf(nestedNameParts[index], currentNestedLevel, value)
	} else if index == len(nestedNameParts) - 2 && isNameAKnownType(nestedNameParts[index+1]) {
		getOrCreateLeaf(nestedNameParts[index], currentNestedLevel, objectifyKnownTypes(value, nestedNameParts[index+1]))
	} else {
		nextLevel := getOrCreateNextLevel(nestedNameParts[index], currentNestedLevel)
		parseNextNestedLevel(index+1, nestedNameParts, nextLevel, value)
	}
}

func getOrCreateLeaf(leafName string, currentNestedLevel exportedMap, value interface{}) {
	if potentialLeaf, exists := currentNestedLevel[leafName]; !exists {
		currentNestedLevel[leafName] = value
	} else {
		nextLevel := potentialLeaf.(exportedMap) // Cannot be otherwise
		nextLevel["Value"] = value
	}
}

func getOrCreateNextLevel(name string, currentLevel exportedMap) exportedMap {
	if potentialNextLevel, exists := currentLevel[name]; exists {
		if nextLevel, ok := potentialNextLevel.(exportedMap); ok {
			return nextLevel
		} else {
			newLevel := make(exportedMap)
			newLevel["Value"] = potentialNextLevel
			currentLevel[name] = newLevel
			return newLevel
		}
	} else {
		newLevel := make(exportedMap)
		currentLevel[name] = newLevel
		return newLevel
	}
}

func isNameAKnownType(name string) bool {
	return name == "Count"  || name == "Number" || name == "Millis" || name == "TimeNano" || name == "Bytes" || name == "Percent" || name == "Seconds"
}

type typedObject struct {
	Type interface {}
	Value interface {}
}

func objectifyKnownTypes(value interface{}, potentialType string) interface{} {
	var actualType interface{}
	if potentialType == "Bytes" || potentialType == "Percent" || potentialType == "Seconds" {
		actualType = potentialType
	} else if potentialType == "Millis" {
		actualType = "Milliseconds"
	} else if potentialType == "TimeNano" {
		actualType = "Nanoseconds"
	}
	if actualType == nil {
		return value
	}
	return &typedObject{Value: value, Type: actualType}
}
