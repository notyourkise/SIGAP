package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCCTVHealthHistoryBeforeCreateGeneratesUUID(t *testing.T) {
	history := CCTVHealthHistory{}

	require.NoError(t, history.BeforeCreate(nil))
	assert.NotEqual(t, uuid.Nil, history.ID)
}

func TestCCTVHealthHistoryBeforeCreatePreservesExistingUUID(t *testing.T) {
	want := uuid.New()
	history := CCTVHealthHistory{ID: want}

	require.NoError(t, history.BeforeCreate(nil))
	assert.Equal(t, want, history.ID)
}
