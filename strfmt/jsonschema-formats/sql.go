package formats

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Scan scans a Date value from database driver type.
func (d *Date) Scan(raw interface{}) error {
	switch v := raw.(type) {
	case []byte:
		return d.UnmarshalText(v)
	case string:
		return d.UnmarshalText([]byte(v))
	case time.Time:
		d.Time = v
		return nil
	case nil:
		*d = Date{}
		return nil
	default:
		return fmt.Errorf("cannot sql.Scan() strfmt.Date from: %#v: %w", v, ErrFormat)
	}
}

// Value converts Date to a primitive value ready to written to a database.
func (d Date) Value() (driver.Value, error) {
	return driver.Value(d.String()), nil
}

// Scan reads a Duration value from database driver type.
func (d *Duration) Scan(raw interface{}) error {
	switch v := raw.(type) {
	case int64:
		d.Duration = time.Duration(v)
	case float64:
		d.Duration = time.Duration(v)
	case nil:
		d.Duration = time.Duration(0)
	case string:
		return d.UnmarshalText([]byte(v))
	case []byte:
		return d.UnmarshalText(v)
	default:
		return fmt.Errorf("cannot sql.Scan() strfmt.Duration from: %#v: %w", v, ErrFormat)
	}

	return nil
}

// Value converts Duration to a primitive value ready to be written to a database.
func (d Duration) Value() (driver.Value, error) {
	return driver.Value(int64(d.Duration)), nil
}

// Scan scans a DateTime value from database driver type.
func (t *DateTime) Scan(raw interface{}) error {
	switch v := raw.(type) {
	case []byte:
		return t.UnmarshalText(v)
	case string:
		return t.UnmarshalText([]byte(v))
	case time.Time:
		t.Time = v
	case nil:
		t.Time = time.Time{}
	default:
		return fmt.Errorf("cannot sql.Scan() strfmt.DateTime from: %#v: %w", v, ErrFormat)
	}

	return nil
}

// Value converts DateTime to a primitive value ready to written to a database.
func (t DateTime) Value() (driver.Value, error) {
	return driver.Value(t.String()), nil
}
