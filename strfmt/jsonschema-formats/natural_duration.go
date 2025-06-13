package formats

import (
	"context"
	"time"
)

// NaturalDuration represents a duration as per the more readable and natural native
// go representation for durations.
//
// Reference: [time.Duration]
type NaturalDuration struct {
	time.Duration
}

func (d NaturalDuration) Validate(_ context.Context) error {
	return nil
}

// MarshalText turns this instance into text
func (d NaturalDuration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText hydrates this instance from text
func (d *NaturalDuration) UnmarshalText(data []byte) error { // validation is performed later on
	dd, err := time.ParseDuration(string(data))
	if err != nil {
		return err
	}
	d.Duration = dd

	return nil
}
