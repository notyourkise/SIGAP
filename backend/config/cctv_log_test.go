package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCCTVStreamLogRetentionDays_Default(t *testing.T) {
	t.Setenv("CCTV_STREAM_LOG_RETENTION_DAYS", "")

	assert.Equal(t, DefaultCCTVStreamLogRetentionDays, CCTVStreamLogRetentionDays())
	assert.Equal(t, 90, CCTVStreamLogRetentionDays())
}

func TestCCTVStreamLogRetentionDays_Configured(t *testing.T) {
	t.Setenv("CCTV_STREAM_LOG_RETENTION_DAYS", "30")
	assert.Equal(t, 30, CCTVStreamLogRetentionDays())

	t.Setenv("CCTV_STREAM_LOG_RETENTION_DAYS", " 45 ")
	assert.Equal(t, 45, CCTVStreamLogRetentionDays())
}

func TestCCTVStreamLogRetentionDays_InvalidFallsBackToDefault(t *testing.T) {
	for _, value := range []string{"0", "-1", "abc", "30.5", "999999999999999999999"} {
		t.Setenv("CCTV_STREAM_LOG_RETENTION_DAYS", value)
		assert.Equal(t, DefaultCCTVStreamLogRetentionDays, CCTVStreamLogRetentionDays(),
			"nilai %q harus jatuh ke default", value)
	}
}
