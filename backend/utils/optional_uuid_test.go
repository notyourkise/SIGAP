package utils

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalUUIDUnmarshalJSON(t *testing.T) {
	t.Run("null eksplisit", func(t *testing.T) {
		var value OptionalUUID
		require.NoError(t, json.Unmarshal([]byte("null"), &value))
		assert.True(t, value.Present)
		assert.True(t, value.Null)
		assert.Nil(t, value.Value)
	})

	t.Run("null dengan spasi", func(t *testing.T) {
		var value OptionalUUID
		require.NoError(t, json.Unmarshal([]byte("  null  "), &value))
		assert.True(t, value.Present)
		assert.True(t, value.Null)
		assert.Nil(t, value.Value)
	})

	t.Run("string kosong", func(t *testing.T) {
		var value OptionalUUID
		require.NoError(t, json.Unmarshal([]byte(`""`), &value))
		assert.True(t, value.Present)
		assert.True(t, value.Null)
	})

	t.Run("UUID valid", func(t *testing.T) {
		id := uuid.New()
		var value OptionalUUID
		require.NoError(t, json.Unmarshal([]byte(`"`+id.String()+`"`), &value))
		assert.True(t, value.Present)
		assert.False(t, value.Null)
		require.NotNil(t, value.Value)
		assert.Equal(t, id, *value.Value)
	})

	t.Run("UUID tidak valid", func(t *testing.T) {
		var value OptionalUUID
		assert.Error(t, json.Unmarshal([]byte(`"bukan-uuid"`), &value))
	})
}
