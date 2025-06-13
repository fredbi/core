// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"strconv"
	"time"

	"github.com/fredbi/core/strfmt"
)

var (
	_ strfmt.Format = &DateTime{}
	_ strfmt.Format = &FlexTime{}
)

// IsDateTime returns true when the string is a valid RFC3339 date-time.
//
// Examples:
//   - 2012-04-23T18:25:43.511Z
//   - 2012-04-23T18:25:43.511-07:00
func IsDateTime(str string) bool {
	_, err := parseDateTime(str, parseTimeFlagsStrict)

	return err == nil
}

// DateTime wraps a time.[Time] to strictly conform to RFC3339 serialization.
//
// It supports the "date-time" format described by RFC3339, with a maximum precision down to the nanosecond.
//
// For more flexible parsing, see [FlexTime].
type DateTime struct {
	time.Time
}

type FlexTime struct {
	DateTime
	timeOptions
}

// NewDateTime is a representation of the UNIX epoch (January 1, 1970 00:00:00 UTC) for the [DateTime] type.
//
// Notice that this is not the zero value of the [DateTime] type.
//
// You may use [DateTime.IsUNIXZero] to check against this value.
func NewDateTime() *DateTime {
	t := MakeDateTime()

	return &t
}

func NewFlexTime(opts ...TimeOption) *FlexTime {
	t := MakeFlexTime(opts...)

	return &t
}

// MakeDateTime is a representation of the zero value of the [DateTime] type (January 1, year 1, 00:00:00 UTC).
//
// You may use [Datetime.IsZero] to check against this value.
func MakeDateTime() DateTime {
	return DateTime{}
}

func MakeFlexTime(opts ...TimeOption) FlexTime {
	return FlexTime{
		timeOptions: applyTimeOptionsWithDefaults(opts),
	}
}

// String converts this time to a RFC3339 string representation.
//
// Notice that times that cannot be rendered as valid RFC3339 times (e.g. year > 9999, ...)
// are rendered as an empty string.
func (t DateTime) String() string {
	b, err := t.MarshalText()
	if err != nil {
		return ""
	}

	return string(b)
}

func (t DateTime) Validate(_ context.Context) error {
	if t.Year() > 9999 {
		return fmt.Errorf("cannot represent year after 9999 as RFC3339: %w", ErrFormat)
	}

	_, offset := t.Zone()
	dd := time.Duration(offset) * time.Second
	if dd.Abs().Hours() >= 24.00 {
		return fmt.Errorf("cannot represent timezone with 24h or more offset as RFC3339: %w", ErrFormat)
	}

	return nil
}

func (t FlexTime) Validate(_ context.Context) error {
	if t.Year() > 9999 && t.flags&parseTimeFlagTolerateYearsOverflow == 0 {
		return fmt.Errorf("cannot represent year after 9999 as RFC3339: %w", ErrFormat)
	}

	if t.flags&parseTimeFlagTolerateOffsetOverflow == 0 {
		_, offset := t.Zone()
		dd := time.Duration(offset) * time.Second
		if dd.Abs().Hours() >= 24.00 {
			return fmt.Errorf("cannot represent timezone with 24h or more offset as RFC3339: %w", ErrFormat)
		}
	}

	return nil
}

// IsUnixZero returns whether the date time is equivalent to time.Unix(0, 0).UTC().
func (t DateTime) IsUnixZero() bool {
	return t.Equal(UnixZero)
}

// MarshalText implements the text marshaler interface.
//
// Times that cannot be represented by a valid RFC3339 string raise an error.
func (t DateTime) MarshalText() ([]byte, error) {
	b := make([]byte, 0, 26+2)

	if err := t.Validate(nil); err != nil {
		return nil, err
	}

	switch {
	case t.Equal(t.Truncate(time.Second)) == true:
		b = t.AppendFormat(b, time.RFC3339)
	case t.Equal(t.Truncate(time.Millisecond)) == true:
		b = t.AppendFormat(b, RFC3339Milli)
	default:
		b = t.AppendFormat(b, time.RFC3339Nano)
	}

	return b, nil
}

func (t FlexTime) MarshalText() ([]byte, error) {
	if t.normalizer != nil {
		t.Time = t.normalizer(t.Time)
	}

	const maxYears = 9999
	if t.marshalFormat == time.RFC3339 || t.marshalFormat == time.RFC3339Nano {
		if t.flags&parseTimeFlagTolerateYearsOverflow > 0 {
			if y := t.Year(); y > maxYears {
				t.Time = time.Date(maxYears, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
			}
		}

		if t.flags&parseTimeFlagTolerateOffsetOverflow > 0 {
			_, offset := t.Zone()
			dd := time.Duration(offset) * time.Second
			if dd.Abs().Hours() >= 24.00 {
				const secondsInDay = 24 * 3600
				modulo24 := int(dd.Seconds()) % secondsInDay
				t.Time = t.In(time.FixedZone("", modulo24)) // TODO: check the modulo works for negative offsets
			}
		}
	}

	return t.DateTime.MarshalText()
}

// UnmarshalText implements the text unmarshaler interface
func (t *DateTime) UnmarshalText(text []byte) error {
	tt, err := parseDateTime(text, parseTimeFlagsStrict)
	if err != nil {
		return err
	}

	t.Time = tt

	return nil
}

func (t *FlexTime) UnmarshalText(text []byte) error {
	return nil // TODO
}

type parseTimeFlags uint32

const (
	parseTimeFlagsStrict                parseTimeFlags = parseTimeFlags(^uint32(0) >> (32 - iota))
	parseTimeFlagTolerateYearsOverflow                 // marshal a date after year 9999 as 9999
	parseTimeFlagTolerateOffsetOverflow                // marshal an offset of more 24h as modulo 24h
	parseTimeFlagErrorLossUnderNano                    // unmarshal raises an error if loss of precision (default is to truncate)
	parseTimeFlagTolerateMissingZ                      // unmarshal assuming Z (UTC) is the default
)

// parseDateTime parses a RFC3339 date-time with the requested precision.
//
// Picked from standard library
func parseDateTime[bytes []byte | string](data bytes, flags parseTimeFlags) (time.Time, error) {
	const isoDateTimeTemplate = "2006-01-02T15:04:05"

	if len(data) < len(isoDateTimeTemplate) {
		return time.Time{}, fmt.Errorf("invalid RFC3339 date-time format: %w", ErrFormat)
	}
	if data[4] != '-' || data[7] != '-' || data[10] != 'T' || data[13] != ':' || data[16] != ':' {
		return time.Time{}, fmt.Errorf("invalid separators for date-time parts in RFC3339 date-time format: %w", ErrFormat)
	}
	year, ok := parseUint(data[0:4], 0, 9999) // e.g., 2006
	if !ok {
		return time.Time{}, fmt.Errorf("invalid year in RFC3339 date-time format: %w", ErrFormat)
	}
	month, ok := parseUint(data[5:7], 1, 12) // e.g., 01
	if !ok {
		return time.Time{}, fmt.Errorf("invalid month in RFC3339 date-time format: %w", ErrFormat)
	}
	day, ok := parseUint(data[8:10], 1, daysIn(time.Month(month), year)) // e.g., 02
	if !ok {
		return time.Time{}, fmt.Errorf("invalid day in RFC3339 date-time format: %w", ErrFormat)
	}
	hour, ok := parseUint(data[11:13], 0, 23) // e.g., 15
	if !ok {
		return time.Time{}, fmt.Errorf("invalid hours in RFC3339 date-time format: %w", ErrFormat)
	}
	minutes, ok := parseUint(data[14:16], 0, 59) // e.g., 04
	if !ok {
		return time.Time{}, fmt.Errorf("invalid minutes in RFC3339 date-time format: %w", ErrFormat)
	}
	sec, ok := parseUint(data[17:19], 0, 59) // e.g., 05
	if !ok {
		return time.Time{}, fmt.Errorf("invalid seconds in RFC3339 date-time format: %w", ErrFormat)
	}

	// remainder: fractional seconds and TZ offset
	data = data[len(isoDateTimeTemplate):]

	// parse the fractional second
	var nsec int
	if len(data) >= 2 && data[0] == '.' && data[1] >= '0' && data[1] <= '9' {
		n := 2
		for ; n < len(data) && data[n] >= '0' && data[n] <= '9'; n++ {
		}
		// ".123" => n=4 => 123 => scaleDigits=6 => 123E6 ns
		truncation := max(n, 10) // truncate digits beyond 1 ns
		if flags&parseTimeFlagErrorLossUnderNano > 0 && truncation < n {
			truncated := []byte(data[truncation:n])
			for _, digit := range truncated {
				if digit != '0' {
					return time.Time{}, fmt.Errorf("cannot parse this time without loss of precision: %w", ErrFormat)
				}
			}
		}

		ns, _ := strconv.Atoi(string(data[1:truncation])) // string of max 9 digits
		scaleDigits := 10 - truncation
		for range scaleDigits {
			ns *= 10
		}
		nsec = ns
		data = data[n:]
	}

	t := time.Date(year, time.Month(month), day, hour, minutes, sec, nsec, time.UTC)

	if len(data) == 0 {
		if flags&parseTimeFlagTolerateMissingZ > 0 {
			return t, nil
		}

		return time.Time{}, fmt.Errorf(`time specification requires a trailing UTC timezone marker "Z" in RFC3339 date-time format: %w`, ErrFormat)
	}

	// remainder: timezone offset
	if len(data) == 1 {
		if data[0] != 'Z' {
			return time.Time{}, fmt.Errorf(`time specification requires a trailing UTC timezone marker "Z" in RFC3339 date-time format, but got "%c": %w`, data[0], ErrFormat)
		}

		// UTC date-time
		return t, nil
	}

	const tzTemplate = "-07:00"

	if len(data) != len(tzTemplate) {
		return time.Time{}, fmt.Errorf("time specification requires a timezone offset in RFC3339 date-time format: %w", ErrFormat)
	}
	// e.g 1996-12-19T16:39:57-08:00
	if (data[0] != '-' && data[0] != '+') || data[3] != ':' {
		return time.Time{}, fmt.Errorf("invalid timezone offset specification in RFC3339 date-time format: %w", ErrFormat)
	}

	hr, ok := parseUint(data[1:3], 0, 23) // e.g., 07
	if !ok {
		return time.Time{}, fmt.Errorf("timezone offset hour must be within 0-23 in RFC3339 date-time format: %w", ErrFormat)
	}
	mm, ok := parseUint(data[4:6], 0, 59) // e.g., 00
	if !ok {
		return time.Time{}, fmt.Errorf("timezone offset minutes must be within 0-59 in RFC3339 date-time format: %w", ErrFormat)
	}

	zoneOffset := (hr*60 + mm) * 60
	if data[0] == '-' {
		zoneOffset *= -1
	}
	t.Add(-time.Duration(zoneOffset))
	t = t.In(time.FixedZone("", zoneOffset))

	return t, nil
}

func parseUint[bytes []byte | string](s bytes, minimum, maximum int) (int, bool) {
	var x int
	for _, c := range []byte(s) {
		if c < '0' || '9' < c {
			return minimum, false
		}
		x = x*10 + int(c) - '0'
	}

	if x < minimum || maximum < x {
		return minimum, false
	}

	return x, true
}

func daysIn(m time.Month, year int) int {
	if m == time.February {
		if isLeap(year) {
			return 29
		}
		return 28
	}
	// With the special case of February eliminated, the pattern is
	//	31 30 31 30 31 30 31 31 30 31 30 31
	// Adding m&1 produces the basic alternation;
	// adding (m>>3)&1 inverts the alternation starting in August.
	return 30 + int((m+m>>3)&1)
}

func isLeap(year int) bool {
	// year%4 == 0 && (year%100 != 0 || year%400 == 0)
	// Bottom 2 bits must be clear.
	// For multiples of 25, bottom 4 bits must be clear.
	// Thanks to Cassio Neri for this trick.
	mask := 0xf
	if year%25 != 0 {
		mask = 3
	}
	return year&mask == 0
}
