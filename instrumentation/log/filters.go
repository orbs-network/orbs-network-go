package log

import "regexp"

type Filter interface {
	Allows(level string, message string, fields []*Field) bool
}

func ExcludeField(field *Field) Filter {
	return &excludeField{field: field}
}

func IncludeFieldWithKey(key string) Filter {
	return &includeFieldWithKey{key: key}
}

func OnlyErrors() Filter {
	return &levelMatch{level: "error"}
}

func IgnoreMessagesMatching(pattern string) Filter {
	return &messageRegexp{pattern: pattern}
}

func IgnoreErrorsMatching(pattern string) Filter {
	return &errorRegexp{pattern: pattern}
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

type levelMatch struct {
	level string
}

func (f *levelMatch) Allows(level string, message string, fields []*Field) bool {
	return level == f.level
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
		if p.Equal(f.field) {
			return false
		}
	}

	return true
}



