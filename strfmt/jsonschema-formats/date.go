// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package formats

import (
	"context"
	"fmt"
	"time"

	"github.com/fredbi/core/strfmt"
)

// TODO: RFC5322 date (net/email)

var _ strfmt.Format = &Date{}

// IsDate tells if a string is a valid ISO8601 date representation.
func IsDate(str string) bool {
	_, err := time.Parse(time.DateOnly, str)

	return err == nil
}

// Date represents a date formatted as per ISO8601 and RFC3339 "full-date".
//
// [Date] extends the standard [time.Time] and inherits all its methods.
//
// Notice that the underlying [time.Time] is always UTC.
//
// See [FlexDate] to support alternate formats and more flexible settings.
//
// References:
//
//   - https://en.wikipedia.org/wiki/ISO_8601
//   - https://datatracker.ietf.org/doc/html/rfc3339.html#anchor14
type Date struct {
	time.Time
}

// TODO: options to set zero at the unix Epoch

func MakeDate() Date {
	return Date{}
}

func NewDate() *Date {
	return &Date{}
}

// String representation of a [Date].
func (d Date) String() string {
	return d.Format(time.DateOnly)
}

func (d Date) Validate(_ context.Context) error {
	// since the inner time may be set independently, we need to run extra validation
	if d.Compare(d.Truncate(24*time.Hour)) != 0 {
		return fmt.Errorf("invalid date: %v: %w", d.Time.String(), ErrFormat)
	}
	if _, offset := d.Zone(); offset != 0 {
		return fmt.Errorf("invalid timezone for date. Requires UTC time: %w", ErrFormat)
	}

	return nil
}

// UnmarshalText parses a text representation into a date type
func (d *Date) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	dd, err := time.ParseInLocation(time.DateOnly, string(text), time.UTC)
	if err != nil {
		return err
	}

	d.Time = dd

	return nil
}

// MarshalText serializes this date type to string
func (d Date) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// Coerce any date built from a [time.Time] into a date that passes validation.
func (d *Date) Coerce() {
	d.Time = d.In(time.UTC).Truncate(24 * time.Hour)
}
