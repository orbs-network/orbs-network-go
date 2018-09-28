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
		{"IncludeParamWithKeyAllowsExpectedKey", IncludeFieldWithKey("foo"), "", "", []*Field{String("foo", "")}, true},
		{"IncludeParamWithKeyRejectsWhenExpectedKeyNotFound", IncludeFieldWithKey("foo"), "", "", nil,false},
		{"OnlyErrorsRejectsInfo", OnlyErrors(), "info", "", nil,false},
		{"OnlyErrorsAllowsError", OnlyErrors(), "error", "", nil,true},
		{"IgnoreMessagesMatchingRejectMessageMatching", IgnoreMessagesMatching("foo.*"), "", "food", nil,false},
		{"IgnoreMessagesMatchingAllowsMismatchingMessages", IgnoreMessagesMatching("food"), "", "foo", nil,true},
		{"IgnoreErrorsMatchingRejectMessageMatching", IgnoreErrorsMatching("foo.*"), "", "", []*Field{Error(errors.Errorf("food"))},false},
		{"IgnoreErrorsMatchingAllowsMismatchingMessages", IgnoreErrorsMatching("food"), "", "", []*Field{Error(errors.Errorf("foo"))},true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.shouldAllow, test.filter.Allows(test.level, test.message, test.params), "test %s did not return expected Allows value", test.name)
		})
	}
}

