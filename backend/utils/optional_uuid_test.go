package utils

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOptionalUUIDUnmarshalJSON(t *testing.T) {
	t.Run("null clears value", func(t *testing.T) {
		var v OptionalUUID
		if err := json.Unmarshal([]byte(`null`), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.Present || !v.Null || v.Value != nil || v.HasValue() {
			t.Fatalf("unexpected state: %+v", v)
		}
	})

	t.Run("empty string clears value", func(t *testing.T) {
		var v OptionalUUID
		if err := json.Unmarshal([]byte(`""`), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.Present || !v.Null || v.Value != nil || v.HasValue() {
			t.Fatalf("unexpected state: %+v", v)
		}
	})

	t.Run("valid uuid", func(t *testing.T) {
		want := uuid.MustParse("11111111-1111-4111-8111-111111111111")
		var v OptionalUUID
		if err := json.Unmarshal([]byte(`"11111111-1111-4111-8111-111111111111"`), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v.Present || v.Null || !v.HasValue() || v.Value == nil || *v.Value != want {
			t.Fatalf("unexpected state: %+v", v)
		}
	})

	t.Run("invalid uuid returns descriptive error", func(t *testing.T) {
		var v OptionalUUID
		err := json.Unmarshal([]byte(`"not-a-uuid"`), &v)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "bukan UUID yang valid") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non-string value returns descriptive error", func(t *testing.T) {
		var v OptionalUUID
		err := json.Unmarshal([]byte(`123`), &v)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "harus berupa string UUID atau null") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing field stays absent", func(t *testing.T) {
		var v OptionalUUID
		if !v.IsZero() || v.Present || v.Null || v.HasValue() {
			t.Fatalf("unexpected state: %+v", v)
		}
	})

	t.Run("null resets previous value", func(t *testing.T) {
		var v OptionalUUID
		if err := json.Unmarshal([]byte(`"11111111-1111-4111-8111-111111111111"`), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := json.Unmarshal([]byte(`null`), &v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.Value != nil || !v.Null || !v.Present || v.HasValue() {
			t.Fatalf("unexpected state: %+v", v)
		}
	})
}
