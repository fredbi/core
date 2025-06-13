package formats

import "time"

const (
	// marshalFormat sets the time resolution format used for marshaling time (set to milliseconds)
	defaultMarshalFormat = RFC3339Milli
)

const (
	// RFC3339Milli represents a ISO8601 format to millis instead of to nanos
	RFC3339Milli = "2006-01-02T15:04:05.000Z07:00"
	// RFC3339MilliNoColon represents a ISO8601 format to millis instead of to nanos
	RFC3339MilliNoColon = "2006-01-02T15:04:05.000Z0700"
	// RFC3339Micro represents a ISO8601 format to micro instead of to nano
	RFC3339Micro = "2006-01-02T15:04:05.000000Z07:00"
	// RFC3339MicroNoColon represents a ISO8601 format to micro instead of to nano
	RFC3339MicroNoColon = "2006-01-02T15:04:05.000000Z0700"
	// ISO8601LocalTime represents a ISO8601 format to ISO8601 in local time (no timezone)
	ISO8601LocalTime = "2006-01-02T15:04:05"
	// ISO8601TimeWithReducedPrecision represents a ISO8601 format with reduced precision (dropped secs)
	ISO8601TimeWithReducedPrecision = "2006-01-02T15:04Z"
	// ISO8601TimeWithReducedPrecisionLocaltime represents a ISO8601 format with reduced precision and no timezone (dropped seconds + no timezone)
	ISO8601TimeWithReducedPrecisionLocaltime = "2006-01-02T15:04"
	// ISO8601TimeUniversalSortableDateTimePattern represents a ISO8601 universal sortable date time pattern.
	ISO8601TimeUniversalSortableDateTimePattern = "2006-01-02 15:04:05"
	// short form of ISO8601TimeUniversalSortableDateTimePattern
	ISO8601TimeUniversalSortableDateTimePatternShortForm = "2006-01-02"
)

var (
	//rxDateTime = regexp.MustCompile(DateTimePattern)

	// DateTimeFormats is the collection of formats used by ParseDateTime()
	defaultDateTimeFormats = []string{RFC3339Micro, RFC3339MicroNoColon, RFC3339Milli, RFC3339MilliNoColon, time.RFC3339, time.RFC3339Nano, ISO8601LocalTime, ISO8601TimeWithReducedPrecision, ISO8601TimeWithReducedPrecisionLocaltime, ISO8601TimeUniversalSortableDateTimePattern, ISO8601TimeUniversalSortableDateTimePatternShortForm}

	// NormalizeTimeForMarshal provides a normalization function on time before marshalling (e.g. time.UTC).
	// By default, the time value is not changed.
	NormalizeTimeForMarshal = func(t time.Time) time.Time { return t }

	// DefaultTimeLocation provides a location for a time when the time zone is not encoded in the string (ex: ISO8601 Local variants).
	DefaultTimeLocation = time.UTC
)

var (
	// UnixZero sets the zero unix UTC timestamp we want to compare against.
	//
	// Unix 0 for an EST timezone is not equivalent to a UTC timezone.
	UnixZero = time.Unix(0, 0).UTC()
)

type TimeOption func(*timeOptions)

type timeOptions struct {
	marshalFormat    string         // TODO: should not be necessary
	supportedFormats []string       // TODO: should not be necessary
	zeroValue        time.Time      // to be seen. Likely more idiomatic to use "default: 1970-01-01T00:00:00Z"
	location         *time.Location // should not be necessary
	normalizer       func(time.Time) time.Time
	flags            parseTimeFlags
}

func applyTimeOptionsWithDefaults(opts []TimeOption) timeOptions {
	o := timeOptions{
		marshalFormat:    defaultMarshalFormat,
		supportedFormats: defaultDateTimeFormats,
		location:         time.UTC,
	}

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

func WithMarshalFormat(format string) TimeOption {
	return func(o *timeOptions) {
		if format != "" {
			o.marshalFormat = format
		}
	}
}
