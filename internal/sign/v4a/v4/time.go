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

import "time"

// SigningTime provides a wrapper around a time.Time which provides cached values for SigV4 signing.
type SigningTime struct {
	time.Time
	timeFormat      string
	shortTimeFormat string
}

// NewSigningTime creates a new SigningTime given a time.Time
func NewSigningTime(t time.Time) SigningTime {
	return SigningTime{
		Time: t,
	}
}

// TimeFormat provides a time formatted in the X-Amz-Date format.
func (m *SigningTime) TimeFormat() string {
	return m.format(&m.timeFormat, TimeFormat)
}

// ShortTimeFormat provides a time formatted of 20060102.
func (m *SigningTime) ShortTimeFormat() string {
	return m.format(&m.shortTimeFormat, ShortTimeFormat)
}

func (m *SigningTime) format(target *string, format string) string {
	if len(*target) > 0 {
		return *target
	}
	v := m.Time.Format(format)
	*target = v
	return v
}
