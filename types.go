package okx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Number is a decimal value kept exactly as OKX sends it, so prices and sizes
// never lose precision. Convert with Float64 for math or feed String into a
// decimal library for money arithmetic.
type Number string

// NumberFromFloat formats f with the minimal number of digits that round-trip.
func NumberFromFloat(f float64) Number {
	return Number(strconv.FormatFloat(f, 'f', -1, 64))
}

// NumberFromInt formats i as a Number.
func NumberFromInt(i int64) Number {
	return Number(strconv.FormatInt(i, 10))
}

func (n Number) String() string { return string(n) }

// Float64 returns the value as float64, or 0 when empty or malformed.
func (n Number) Float64() float64 {
	f, _ := strconv.ParseFloat(string(n), 64)
	return f
}

// Int64 returns the value as int64, or 0 when empty or not an integer.
func (n Number) Int64() int64 {
	i, _ := strconv.ParseInt(string(n), 10, 64)
	return i
}

// IsZero reports whether the value is empty or numerically zero.
func (n Number) IsZero() bool { return n == "" || n.Float64() == 0 }

func (n *Number) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		*n = ""
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*n = Number(s)
		return nil
	}
	*n = Number(b)
	return nil
}

// Time is a timestamp that OKX encodes as Unix milliseconds, either as a
// string or a number. An empty value decodes to the zero time.
type Time struct{ time.Time }

// TimeFromMillis converts Unix milliseconds to Time.
func TimeFromMillis(ms int64) Time { return Time{time.UnixMilli(ms)} }

func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + strconv.FormatInt(t.UnixMilli(), 10) + `"`), nil
}

func (t *Time) UnmarshalJSON(b []byte) error {
	s := string(bytes.Trim(b, `"`))
	if s == "" || s == "null" || s == "0" {
		t.Time = time.Time{}
		return nil
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.UnixMilli(ms)
	return nil
}

// Bool decodes booleans that OKX sometimes sends as "true"/"false" strings.
type Bool bool

func (v *Bool) UnmarshalJSON(b []byte) error {
	switch string(bytes.Trim(b, `"`)) {
	case "true", "1":
		*v = true
	default:
		*v = false
	}
	return nil
}

// Int decodes integers that OKX sends either as strings or numbers.
type Int int64

func (v *Int) UnmarshalJSON(b []byte) error {
	s := string(bytes.Trim(b, `"`))
	if s == "" || s == "null" {
		*v = 0
		return nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*v = Int(i)
	return nil
}

func unmarshalArray(b []byte, dst []string) error {
	var f []string
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	if len(f) < len(dst) {
		return fmt.Errorf("okx: expected %d fields, got %d", len(dst), len(f))
	}
	copy(dst, f)
	return nil
}
