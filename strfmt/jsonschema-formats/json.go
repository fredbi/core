package formats

import (
	"encoding/json"
	"slices"
	"time"
)

// MarshalJSON returns the Date as JSON
func (d Date) MarshalJSON() ([]byte, error) {
	const dateSize = 10

	b := make([]byte, 0, dateSize)
	b = append(b, '"')
	d.AppendFormat(b, time.DateOnly)
	b = append(b, '"')

	return b, nil
}

// UnmarshalJSON sets the Date from JSON
func (d *Date) UnmarshalJSON(data []byte) error {
	var strdate string
	if err := json.Unmarshal(data, &strdate); err != nil {
		return err
	}

	return d.UnmarshalText([]byte(strdate))
}

// MarshalJSON returns a [Duration] as JSON
func (d Duration) MarshalJSON() ([]byte, error) {
	b, err := d.MarshalText()
	if err != nil {
		return b, err
	}
	slices.Grow(b, len(b)+2)

	b = slices.Insert(b, 0, '"')
	b = append(b, '"')

	return b, nil
}

// UnmarshalJSON sets a [Duration] from JSON
func (d *Duration) UnmarshalJSON(data []byte) error {
	var dstr string
	if err := json.Unmarshal(data, &dstr); err != nil {
		return err
	}
	dd, err := parseDuration([]byte(dstr))
	if err != nil {
		return err
	}
	d.Duration = dd

	return nil
}

func (t DateTime) MarshalJSON() ([]byte, error) {
	b, err := t.MarshalText()
	if err != nil {
		return nil, err
	}

	slices.Grow(b, len(b)+2)
	b = slices.Insert(b, 0, '"')
	b = append(b, '"')

	return b, nil
}

// MarshalJSON returns the DateTime as JSON
func (t FlexTime) MarshalJSON() ([]byte, error) {
	b, err := t.MarshalText()
	if err != nil {
		return nil, err
	}

	slices.Grow(b, len(b)+2)
	b = slices.Insert(b, 0, '"')
	b = append(b, '"')

	return b, nil
}

// UnmarshalJSON sets the DateTime from JSON
func (t *FlexTime) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil // maybe error?
	}

	// TODO: use my own lexer and get rid of the JSON stdlib
	var tstr string
	if err := json.Unmarshal(data, &tstr); err != nil {
		return err
	}

	tt, err := parseDateTime(tstr, parseTimeFlagsStrict)
	if err != nil {
		return err
	}
	t.Time = tt

	return nil
}

// TODO: MarshalJSONValidate(context.Context) ([]byte,error), UnmarshalJSONValidate(context.Context, []byte) error
// TODO: EncodeJSONValidate(context.Context, io.Writer) error, DecodeJSONValidate(context.Context, io.Reader) error
