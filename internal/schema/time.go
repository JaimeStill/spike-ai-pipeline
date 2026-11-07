package schema

import (
	"fmt"
	"time"
)

// Version is the schema's version, which every run manifest names.
const Version = "1"

// timeLayout is RFC 3339 with nine digits of fraction, always written, so every time in the
// schema has the same width.
const timeLayout = "2006-01-02T15:04:05.000000000Z07:00"

// Time is a time as the schema writes it: a JSON string, RFC 3339, in UTC, with nine digits of
// nanoseconds, such as "2026-11-07T15:04:05.123456789Z". A field of it marked optional is absent
// when the time is zero.
type Time time.Time

// IsZero reports whether t is the zero time, which an optional field leaves out.
func (t Time) IsZero() bool { return time.Time(t).IsZero() }

// String returns t as the schema writes it.
func (t Time) String() string { return time.Time(t).UTC().Format(timeLayout) }

// MarshalJSON returns t as a JSON string, as [Time.String] writes it.
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

// UnmarshalJSON reads t from a JSON string in RFC 3339, with any number of fraction digits.
func (t *Time) UnmarshalJSON(b []byte) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return fmt.Errorf("schema: time %s is not a JSON string", b)
	}
	parsed, err := time.Parse(time.RFC3339Nano, string(b[1:len(b)-1]))
	if err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	*t = Time(parsed.UTC())
	return nil
}
