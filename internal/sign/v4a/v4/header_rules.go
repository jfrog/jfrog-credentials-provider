// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v4

import (
	"strings"
)

// Rules houses a set of Rule needed for validation of a
// string value
type Rules []Rule

// Rule interface allows for more flexible rules and just simply
// checks whether or not a value adheres to that Rule
type Rule interface {
	IsValid(value string) bool
}

// IsValid will iterate through all rules and see if any rules
// apply to the value and supports nested rules
func (r Rules) IsValid(value string) bool {
	for _, rule := range r {
		if rule.IsValid(value) {
			return true
		}
	}
	return false
}

// MapRule generic Rule for maps
type MapRule map[string]struct{}

// IsValid for the map Rule satisfies whether it exists in the map
func (m MapRule) IsValid(value string) bool {
	_, ok := m[value]
	return ok
}

// AllowList is a generic Rule for whitelisting
type AllowList struct {
	Rule
}

// IsValid for AllowList checks if the value is within the AllowList
func (w AllowList) IsValid(value string) bool {
	return w.Rule.IsValid(value)
}

// DenyList is a generic Rule for blacklisting
type DenyList struct {
	Rule
}

// IsValid for AllowList checks if the value is within the AllowList
func (b DenyList) IsValid(value string) bool {
	return !b.Rule.IsValid(value)
}

// Patterns is a list of strings to match against
type Patterns []string

// IsValid for Patterns checks each pattern and returns if a match has
// been found
func (p Patterns) IsValid(value string) bool {
	for _, pattern := range p {
		if hasPrefixFold(value, pattern) {
			return true
		}
	}
	return false
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[0:len(prefix)], prefix)
}

// InclusiveRules rules allow for rules to depend on one another
type InclusiveRules []Rule

// IsValid will return true if all rules are true
func (r InclusiveRules) IsValid(value string) bool {
	for _, rule := range r {
		if !rule.IsValid(value) {
			return false
		}
	}
	return true
}
