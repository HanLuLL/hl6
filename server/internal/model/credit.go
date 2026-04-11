package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// Credit represents credit amounts internally as integers.
// 1 Credit unit = 0.1 display credits. So 15 = 1.5 credits displayed.
type Credit int64

const (
	// CreditScale means 1 displayed credit equals 10 internal units.
	CreditScale int64 = 10
	// MaxDisplayCredit is the absolute max allowed for a single credit input.
	MaxDisplayCredit = 100000.0
	// MaxCredit is the internal-unit upper bound for a single credit input.
	MaxCredit Credit = 1000000
)

var (
	ErrInvalidCreditNumber = errors.New("invalid credit number")
	ErrCreditOutOfRange    = errors.New("credit out of range")
	ErrCreditPrecision     = errors.New("credit precision exceeds one decimal place")
	ErrCreditNegative      = errors.New("credit must be non-negative")
)

// CreditFromFloat converts a display float (e.g., 1.5) to internal Credit (15).
func CreditFromFloat(f float64) Credit {
	return Credit(math.Round(f * float64(CreditScale)))
}

// ToFloat converts internal Credit to display float.
func (c Credit) ToFloat() float64 {
	return float64(c) / float64(CreditScale)
}

// ParseDisplayCredit validates and converts a display credit amount into internal units.
// When allowNegative is false, only non-negative values are accepted.
// When requireOneDecimal is true, only 0.1 precision is accepted.
func ParseDisplayCredit(f float64, allowNegative bool, requireOneDecimal bool) (Credit, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, ErrInvalidCreditNumber
	}
	if !allowNegative && f < 0 {
		return 0, ErrCreditNegative
	}
	if math.Abs(f) > MaxDisplayCredit {
		return 0, ErrCreditOutOfRange
	}
	if requireOneDecimal {
		scaled := f * float64(CreditScale)
		if math.Abs(scaled-math.Round(scaled)) > 1e-9 {
			return 0, ErrCreditPrecision
		}
	}
	return CreditFromFloat(f), nil
}

// MarshalJSON outputs Credit as a float for JSON (e.g., 15 -> 1.5).
func (c Credit) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ToFloat())
}

// UnmarshalJSON parses a float from JSON into Credit.
func (c *Credit) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	*c = CreditFromFloat(f)
	return nil
}

// Scan implements sql.Scanner for reading from the database.
func (c *Credit) Scan(value interface{}) error {
	if value == nil {
		*c = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*c = Credit(v)
		return nil
	case float64:
		*c = Credit(math.Round(v))
		return nil
	default:
		return fmt.Errorf("cannot scan %T into Credit", value)
	}
}

// Value implements driver.Valuer for writing to the database.
func (c Credit) Value() (driver.Value, error) {
	return int64(c), nil
}
