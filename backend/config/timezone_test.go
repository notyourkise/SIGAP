package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppLocation_DefaultsToMakassar(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "")

	location := AppLocation()
	require.NotNil(t, location)
	assert.Equal(t, DefaultAppTimezone, location.String())

	_, offset := time.Date(2026, 10, 27, 12, 0, 0, 0, location).Zone()
	assert.Equal(t, 8*60*60, offset, "WITA harus UTC+8")
}

func TestAppLocation_UsesConfiguredTimezone(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Asia/Jakarta")

	location := AppLocation()
	require.NotNil(t, location)
	assert.Equal(t, "Asia/Jakarta", location.String())

	_, offset := time.Date(2026, 10, 27, 12, 0, 0, 0, location).Zone()
	assert.Equal(t, 7*60*60, offset, "WIB harus UTC+7")
}

func TestAppLocation_InvalidFallsBackToWITA(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Bukan/Zona/Waktu")

	location := AppLocation()
	require.NotNil(t, location)
	assert.Equal(t, "WITA", location.String())

	_, offset := time.Date(2026, 10, 27, 12, 0, 0, 0, location).Zone()
	assert.Equal(t, 8*60*60, offset)
}
