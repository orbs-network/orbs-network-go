// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"regexp"
)

type Filter interface {
	Allows(level string, message string, fields []*Field) bool
}

type ConditionalFilter interface {
	Filter
	On()
	Off()
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
	compiledPattern, _ := regexp.Compile(pattern)
	return &messageRegexp{
		pattern:         pattern,
		compiledPattern: compiledPattern,
	}
}

func IgnoreErrorsMatching(pattern string) Filter {
	compiledPattern, _ := regexp.Compile(pattern)
	return &errorRegexp{pattern: pattern,
		compiledPattern: compiledPattern,
	}
}

func DiscardAll() Filter {
	return &discardAll{}
}

func OnlyMetrics() Filter {
	return &onlyMetrics{}
}

type errorRegexp struct {
	pattern         string
	compiledPattern *regexp.Regexp
}

func (f *errorRegexp) Allows(level string, message string, fields []*Field) bool {
	for _, field := range fields {
		if field.Type == ErrorType {
			if f.compiledPattern == nil {
				return false
			}
			if f.compiledPattern.MatchString(field.Error.Error()) {
				return false
			}
		}
	}

	return true
}

type messageRegexp struct {
	pattern         string
	compiledPattern *regexp.Regexp
}

func (f *messageRegexp) Allows(level string, message string, fields []*Field) bool {
	if f.compiledPattern == nil {
		return false
	}
	return !f.compiledPattern.MatchString(message)
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

type conditionalFilter struct {
	enabled bool
	filter  Filter
}

func NewConditionalFilter(enabled bool, filter Filter) ConditionalFilter {
	return &conditionalFilter{enabled, filter}
}

func (f *conditionalFilter) On() {
	f.enabled = true
}

func (f *conditionalFilter) Off() {
	f.enabled = false
}

func (f *conditionalFilter) Allows(level string, message string, fields []*Field) bool {
	if f.enabled && f.filter != nil {
		return f.filter.Allows(level, message, fields)
	}

	return true
}
