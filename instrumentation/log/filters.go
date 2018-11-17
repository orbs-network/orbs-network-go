package log

import (
	"regexp"
)

type Filter interface {
	Allows(level string, message string, fields []*Field) bool
}

func ExcludeEntryPoint(name string) Filter {
	return ExcludeField(String("entry-point", name))
}

func ExcludeField(field *Field) Filter {
	return &excludeField{field: field}
}

func IncludeFieldWithKey(key string) Filter {
	return &includeFieldWithKey{key: key}
}

func Or(filters ...Filter) Filter {
	return &or{filters}
}

// FIXME add failed expectations here
func OnlyErrors() Filter {
	return &onlyErrors{}
}

func OnlyCheckpoints() Filter {
	return &matchField{String("flow", "checkpoint")}
}

func MatchField(f *Field) Filter {
	return &matchField{f}
}

func IgnoreMessagesMatching(pattern string) Filter {
	return &messageRegexp{pattern: pattern}
}

func IgnoreErrorsMatching(pattern string) Filter {
	return &errorRegexp{pattern: pattern}
}

func DiscardAll() Filter {
	return &discardAll{}
}

func OnlyMetrics() Filter {
	return &onlyMetrics{}
}

type errorRegexp struct {
	pattern string
}

func (f *errorRegexp) Allows(level string, message string, fields []*Field) bool {
	for _, field := range fields {
		if field.Type == ErrorType {

			if match, _ := regexp.MatchString(f.pattern, field.Error.Error()); match {
				return false
			}
		}
	}

	return true
}

type messageRegexp struct {
	pattern string
}

func (f *messageRegexp) Allows(level string, message string, fields []*Field) bool {
	match, _ := regexp.MatchString(f.pattern, message)
	return !match
}

type onlyErrors struct {
}

func (f *onlyErrors) Allows(level string, message string, fields []*Field) bool {
	if level == "error" {
		return true
	}

	for _, f := range fields {
		if f.Type == ErrorType {
			return true
		}
	}

	return false
}

type includeFieldWithKey struct {
	key string
}

func (f *includeFieldWithKey) Allows(level string, message string, fields []*Field) bool {
	for _, p := range fields {
		if p.Key == f.key {
			return true
		}
	}

	return false
}

type excludeField struct {
	field *Field
}

func (f *excludeField) Allows(level string, message string, fields []*Field) bool {
	for _, p := range fields {
		if p.IsNested() {
			return f.Allows(level, message, p.Nested.NestedFields())
		}
		if p.Equal(f.field) {
			return false
		}

	}

	return true
}

type matchField struct {
	field *Field
}

func (f *matchField) Allows(level string, message string, fields []*Field) bool {
	for _, p := range fields {
		if p.Equal(f.field) {
			return true
		}
	}

	return false
}

type or struct {
	filters []Filter
}

func (f *or) Allows(level string, message string, fields []*Field) bool {
	result := false

	for _, f := range f.filters {
		result = result || f.Allows(level, message, fields)
	}

	return result
}

type discardAll struct {
}

func (discardAll) Allows(level string, message string, fields []*Field) bool {
	return false
}

type onlyMetrics struct {
}

func (f onlyMetrics) Allows(level string, message string, fields []*Field) bool {
	return level == "metric"
}
