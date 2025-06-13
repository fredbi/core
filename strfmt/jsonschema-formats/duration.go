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
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/fredbi/core/strfmt"
)

const (
	dayDuration   = 24 * time.Hour
	weekDuration  = 7 * dayDuration
	yearDuration  = 365 * dayDuration
	monthDuration = 30 * dayDuration
)

var (
	_ strfmt.Format = &Duration{}
	_ strfmt.Format = &NaturalDuration{}
)

// IsDuration tells if the provided string is a valid RFC3339 duration.
func IsDuration(str string) bool {
	_, err := parseDuration([]byte(str))

	return err == nil
}

// Duration represents a duration as per ISO8601
// and RFC3339.
//
// Notice that the ISO8601 duration format may define durations only down to the second.
//
// Examples:
//
//   - P1Y2M3DT4H5M6S
//   - P1W
//
// [Duration] is a wrapper around [time.Duration] and inherits its methods.
//
// A [Duration], like [time.Duration], internally stores a period of time as a nanosecond count, with the largest
// representable duration being approximately 290 years. Well-formed ISO8601 durations that overflow this limit
// are not considered valid [Duration] s.
//
// Attention: RFC3339 durations differ significantly from natively parsed go durations. Also RFC3339 durations are
// always positive.
//
// Calendar years have 365 days.
//
// Calendar months have 30 days.
//
// See [NaturalDuration] to support the native go syntax for durations (e.g. 1h2m3s).
// See [FlexDuration] to support more formats and settings.
//
// References:
//
//   - https://en.wikipedia.org/wiki/ISO_8601
//   - https://datatracker.ietf.org/doc/html/rfc3339.html#appendix-A
type Duration struct {
	// notes: Java standard duration supports negative durations and fractional parts
	time.Duration
}

func (d Duration) String() string {
	b, _ := d.MarshalText()

	return string(b)
}

func (d Duration) MarshalText() ([]byte, error) {
	if d.Duration == 0 {
		return []byte{'P', '0', 'D'}, nil
	}

	var buf bytes.Buffer
	buf.WriteByte('P')

	daysPart := int64(d.Truncate(dayDuration).Hours())
	days := daysPart / 24
	if days > 0 {
		buf.WriteString(strconv.FormatInt(days, 10))
		buf.WriteByte('D')
	}

	remainder := d.Duration - time.Duration(daysPart)
	if remainder > 0 {
		buf.WriteByte('T')
	}

	hours := int64(remainder.Truncate(time.Hour).Hours())
	if hours > 0 {
		buf.WriteString(strconv.FormatInt(hours, 10))
		buf.WriteByte('H')
	}

	remainder -= time.Duration(hours) * time.Hour
	minutes := int64(remainder.Truncate(time.Minute))
	if minutes > 0 {
		buf.WriteString(strconv.FormatInt(minutes, 10))
		buf.WriteByte('M')
	}

	remainder -= time.Duration(minutes) * time.Minute
	seconds := int64(remainder.Truncate(time.Second))
	if seconds > 0 {
		buf.WriteString(strconv.FormatInt(seconds, 10))
		buf.WriteByte('S')
	}

	remainder -= time.Duration(seconds) * time.Second
	if remainder > 0 {
		return buf.Bytes(), fmt.Errorf("cannot represent a duration with fractional seconds as a ISO8601 duration: %w", ErrFormat)
	}

	return buf.Bytes(), nil
}

func (d Duration) Validate(_ context.Context) error {
	if d.Duration > 0 && (d.Duration == d.Truncate(time.Second)) { // should we raise an error here?
		return nil
	}

	return fmt.Errorf("duration is invalid: %w", ErrFormat)
}

func (d *Duration) UnmarshalText(data []byte) error { // validation is performed later on
	dd, err := parseDuration(data)
	if err != nil {
		return err
	}

	d.Duration = dd

	return nil
}

// parseRFC3339Duration parses a duration from a string, as per RFC3339
func parseDuration(input []byte) (time.Duration, error) {
	if len(input) < 2 {
		return 0, fmt.Errorf("duration can't be empty: %w", ErrFormat)
	}

	if input[0] != 'P' {
		return 0, fmt.Errorf(`RFC3339 duration must start with "P": %w`, ErrFormat)
	}

	const (
		weekSection int = iota
		yearSection
		monthSection
		daySection
		hourSection
		minuteSection
		secondSection
	)

	var (
		isWeekDuration  bool
		isInTimeSection bool
		currentSection  int
		parts           [7]int64
	)
	currentDigits := make([]byte, 0, 10)

	for index := 1; index < len(input); {
		c := input[index]

		switch c {
		case 'W':
			if len(currentDigits) == 0 {
				return 0, fmt.Errorf(`period designator "%c" must follow digits in string %q: %w"`, c, string(input), ErrFormat)
			}
			if isInTimeSection || currentSection > weekSection {
				return 0, fmt.Errorf(`week period designator "%c" not valid in this context: %q: %w"`, c, string(input), ErrFormat)
			}
			weeks, err := strconv.ParseInt(string(currentDigits), 10, 64)
			if err != nil {
				return 0, fmt.Errorf(`invalid conversion: %w"`, c, string(input), ErrFormat)
			}
			isWeekDuration = true
			parts[weekSection] = weeks
			currentDigits = currentDigits[:0]

		case 'Y':
			if len(currentDigits) == 0 {
				return 0, fmt.Errorf(`period designator "%c" must follow digits in string %q: %w"`, c, string(input), ErrFormat)
			}
			if isWeekDuration || isInTimeSection || currentSection > yearSection {
				return 0, fmt.Errorf(`year period designator "%c" not valid in this context: %q: %w"`, c, string(input), ErrFormat)
			}
			years, err := strconv.ParseInt(string(currentDigits), 10, 64)
			if err != nil {
				return 0, fmt.Errorf(`invalid conversion: %w"`, c, string(input), ErrFormat)
			}
			parts[yearSection] = years
			currentSection = yearSection
			currentDigits = currentDigits[:0]

		case 'M': // month or minute
			if len(currentDigits) == 0 {
				return 0, fmt.Errorf(`period designator "%c" must follow digits in string %q: %w"`, c, string(input), ErrFormat)
			}
			if isInTimeSection {
				if isWeekDuration || currentSection > minuteSection {
					return 0, fmt.Errorf(`minute period designator "%c" not valid in this context: %q: %w"`, c, string(input), ErrFormat)
				}
				minutes, err := strconv.ParseInt(string(currentDigits), 10, 64)
				if err != nil {
					return 0, fmt.Errorf(`invalid conversion: %w"`, c, string(input), ErrFormat)
				}
				parts[minuteSection] = minutes
				currentSection = minuteSection
				continue
			}

			if isWeekDuration || currentSection > monthSection {
				return 0, fmt.Errorf(`month period designator "%c" not valid in this context: %q: %w"`, c, string(input), ErrFormat)
			}
			months, err := strconv.ParseInt(string(currentDigits), 10, 64)
			if err != nil {
				return 0, fmt.Errorf(`invalid conversion: %w"`, c, string(input), ErrFormat)
			}
			parts[monthSection] = months
			currentSection = monthSection
			currentDigits = currentDigits[:0]

		case 'D':
			if len(currentDigits) == 0 {
				return 0, fmt.Errorf(`period designator "%c" must follow digits in string %q: %w"`, c, string(input), ErrFormat)
			}
			if isWeekDuration || isInTimeSection || currentSection > daySection {
				return 0, fmt.Errorf(`day period designator "%c" not valid in this context: %q: %w"`, c, string(input), ErrFormat)
			}
			days, err := strconv.ParseInt(string(currentDigits), 10, 64)
			if err != nil {
				return 0, fmt.Errorf(`invalid conversion: %w"`, c, string(input), ErrFormat)
			}
			parts[daySection] = days
			currentSection = daySection
			currentDigits = currentDigits[:0]

		case 'H':
			if len(currentDigits) == 0 {
				return 0, fmt.Errorf(`period designator "%c" must follow digits in string %q: %w"`, c, string(input), ErrFormat)
			}
			if isWeekDuration || !isInTimeSection || currentSection > hourSection {
				return 0, fmt.Errorf(`hour period designator "%c" not valid in this context: %q: %w"`, c, string(input), ErrFormat)
			}
			hours, err := strconv.ParseInt(string(currentDigits), 10, 64)
			if err != nil {
				return 0, fmt.Errorf(`invalid conversion: %w"`, c, string(input), ErrFormat)
			}
			parts[hourSection] = hours
			currentSection = hourSection
			currentDigits = currentDigits[:0]

		case 'S':
			if len(currentDigits) == 0 {
				return 0, fmt.Errorf(`period designator "%c" must follow digits in string %q: %w"`, c, string(input), ErrFormat)
			}
			if isWeekDuration || !isInTimeSection {
				return 0, fmt.Errorf(`hour period designator "%c" not valid in this context: %q: %w"`, c, string(input), ErrFormat)
			}
			seconds, err := strconv.ParseInt(string(currentDigits), 10, 64)
			if err != nil {
				return 0, fmt.Errorf(`invalid conversion: %w"`, c, string(input), ErrFormat)
			}
			parts[secondSection] = seconds
			currentSection = secondSection
			currentDigits = currentDigits[:0]

		case 'T': // time section
			if len(currentDigits) != 0 {
				return 0, fmt.Errorf(`period designator "%c" must not follow digits in string %q: %w"`, c, string(input), ErrFormat)
			}
			if isWeekDuration || isInTimeSection {
				return 0, fmt.Errorf(`time period designator "%c" not valid in this context: %q: %w"`, c, string(input), ErrFormat)
			}
			isInTimeSection = true

		default:
			for ; index < len(input); index++ {
				// TODO: use parseUint cf. DateTime
				// parse a group of digits
				if c < '0' || c > '9' {
					break
				}
				currentDigits = append(currentDigits, c)

			}

			if index == len(input) {
				return 0, fmt.Errorf(`invalid trailing digit: %q: %w"`, c, string(input), ErrFormat)
			}

			switch input[index] {
			case 'W', 'Y', 'M', 'D', 'H', 'S', 'T':
				// valid period designator. Invalid 'T' will be caught above with the appropriate error
				// message.
				continue
			default:
				return 0, fmt.Errorf(`invalid token "%c" found in duration string %q: %w"`, c, string(input), ErrFormat)
			}
		}
	}

	// concatenate parts

	const errMsg = "duration string exceed the maximum duration supported by time.Duration: %w"
	d, ok := addMultOverflow(time.Duration(0), time.Duration(parts[weekSection]), weekDuration)
	if !ok {
		return 0, fmt.Errorf(errMsg, ErrFormat)
	}
	d, ok = addMultOverflow(d, time.Duration(parts[yearSection]), yearDuration)
	if !ok {
		return 0, fmt.Errorf(errMsg, ErrFormat)
	}
	d, ok = addMultOverflow(d, time.Duration(parts[monthSection]), monthDuration)
	if !ok {
		return 0, fmt.Errorf(errMsg, ErrFormat)
	}
	d, ok = addMultOverflow(d, time.Duration(parts[daySection]), dayDuration)
	if !ok {
		return 0, fmt.Errorf(errMsg, ErrFormat)
	}
	d, ok = addMultOverflow(d, time.Duration(parts[hourSection]), time.Hour)
	if !ok {
		return 0, fmt.Errorf(errMsg, ErrFormat)
	}
	d, ok = addMultOverflow(d, time.Duration(parts[minuteSection]), time.Minute)
	if !ok {
		return 0, fmt.Errorf(errMsg, ErrFormat)
	}
	d, ok = addMultOverflow(d, time.Duration(parts[secondSection]), time.Second)
	if !ok {
		return 0, fmt.Errorf(errMsg, ErrFormat)
	}

	return d, nil
}

// addMult returns a + b *c with an overflow status
func addMultOverflow[T ~int64](a, b, c T) (T, bool) {
	if b == 0 || c == 0 {
		return a, true
	}
	x := b * c
	if (x < 0) == ((b < 0) != (c < 0)) {
		if x/c == b {
			return addOverflow(a, x)
		}
	}

	return x, false
}

func addOverflow[T ~int64](a, b T) (T, bool) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, true
	}
	return c, false
}
