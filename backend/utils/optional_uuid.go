package utils

import (
	"bytes"
	"strings"

	"github.com/google/uuid"
)

// OptionalUUID handles three states when parsing JSON:
//   - Not sent in JSON (Present=false) → keep existing value
//   - Valid UUID string (Present=true, Value=<uuid>) → update to new UUID
//   - Empty string or null (Present=true, Null=true) → set to NULL/database nil
type OptionalUUID struct {
	Value   *uuid.UUID
	Present bool // true if field was explicitly in JSON
	Null    bool // true if field is null or empty string
}

// UnmarshalJSON implements json.Unmarshaler interface
func (o *OptionalUUID) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	str := strings.Trim(string(data), `"`)

	// Handle null or empty string
	if str == "" || bytes.Equal(data, []byte("null")) {
		o.Value = nil
		o.Null = true
		o.Present = true
		return nil
	}

	// Try to parse as UUID
	id, err := uuid.Parse(str)
	if err != nil {
		return err
	}

	o.Value = &id
	o.Present = true
	o.Null = false
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (o OptionalUUID) MarshalJSON() ([]byte, error) {
	if o.Value == nil {
		return []byte("null"), nil
	}
	return []byte("\"" + o.Value.String() + "\""), nil
}

// IsZero returns true if the OptionalUUID has no value (not present or null)
func (o OptionalUUID) IsZero() bool {
	return !o.Present || o.Null
}

// HasValue returns true if the OptionalUUID has a valid UUID value
func (o OptionalUUID) HasValue() bool {
	return o.Present && !o.Null && o.Value != nil
}
