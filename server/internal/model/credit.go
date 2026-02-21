package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
)

// Credit represents credit amounts internally as integers.
// 1 Credit unit = 0.1 display credits. So 15 = 1.5 credits displayed.
type Credit int64

// CreditFromFloat converts a display float (e.g., 1.5) to internal Credit (15).
func CreditFromFloat(f float64) Credit {
	return Credit(math.Round(f * 10))
}

// ToFloat converts internal Credit to display float.
func (c Credit) ToFloat() float64 {
	return float64(c) / 10.0
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
