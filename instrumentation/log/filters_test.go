// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFilters(t *testing.T) {
	tests := []struct {
		name        string
		filter      Filter
		level       string
		message     string
		params      []*Field
		shouldAllow bool
	}{
		{"ExcludeFieldRejectsParam", ExcludeField(Service("foo")), "", "", []*Field{Service("foo")}, false},
		{"ExcludeFieldAllowsOtherParam", ExcludeField(Service("foo")), "", "", []*Field{Service("food")}, true},
		{"ExcludeFieldAllowsWithNoParams", ExcludeField(Service("foo")), "", "", nil, true},
		{"ExcludeFieldHandlesNestedParams", ExcludeField(Service("foo")), "", "", aggregate(Service("foo")), false},
		{"IncludeParamWithKeyAllowsExpectedKey", IncludeFieldWithKey("foo"), "", "", []*Field{String("foo", "")}, true},
		{"IncludeParamWithKeyRejectsWhenExpectedKeyNotFound", IncludeFieldWithKey("foo"), "", "", nil, false},
		{"OnlyErrorsRejectsInfo", OnlyErrors(), "info", "", nil, false},
		{"OnlyErrorsAllowsError", OnlyErrors(), "error", "", nil, true},
		{"OnlyErrorsAllowsInfoWithErrorParam", OnlyErrors(), "info", "", []*Field{Error(errors.Errorf("foo"))}, true},
		{"MatchFieldAllowsField", MatchField(String("hello", "world")), "info", "", []*Field{String("hello", "world")}, true},
		{"MatchFieldRejectsDifferentField", MatchField(String("hello", "world")), "info", "", []*Field{String("hello", "mom")}, false},
		{"OnlyCheckpointsAllowsInfo", OnlyCheckpoints(), "info", "", []*Field{String("flow", "checkpoint")}, true},
		{"OnlyCheckpointsRejectsDifferentField", OnlyCheckpoints(), "info", "", nil, false},
		{"IgnoreMessagesMatchingRejectMessageMatching", IgnoreMessagesMatching("foo.*"), "", "food", nil, false},
		{"IgnoreMessagesMatchingAllowsMismatchingMessages", IgnoreMessagesMatching("food"), "", "foo", nil, true},
		{"IgnoreErrorsMatchingRejectMessageMatching", IgnoreErrorsMatching("foo.*"), "", "", []*Field{Error(errors.Errorf("food"))}, false},
		{"IgnoreErrorsMatchingAllowsMismatchingMessages", IgnoreErrorsMatching("food"), "", "", []*Field{Error(errors.Errorf("foo"))}, true},
		{"OrAllowsErrors", Or(OnlyErrors(), OnlyCheckpoints()), "error", "", nil, true},
		{"OrAllowsCheckpoints", Or(OnlyErrors(), OnlyCheckpoints()), "info", "", []*Field{String("flow", "checkpoint")}, true},
		{"OrRejectsNonErrorNonCheckpoint", Or(OnlyErrors(), OnlyCheckpoints()), "info", "", nil, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.shouldAllow, test.filter.Allows(test.level, test.message, test.params), "test %s did not return expected Allows value", test.name)
		})
	}
}

func aggregate(fields ...*Field) []*Field {
	return []*Field{{Key: "baz", Nested: &aggregateField{fields: fields}, Type: AggregateType}}
}

type aggregateField struct {
	fields []*Field
}

func (f *aggregateField) NestedFields() []*Field {
	return f.fields
}
