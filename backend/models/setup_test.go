package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBaseBeforeCreateGeneratesMissingUUID(t *testing.T) {
	base := Base{}

	assert.NoError(t, base.BeforeCreate(nil))
	assert.NotEqual(t, uuid.Nil, base.ID)
}

func TestBaseBeforeCreatePreservesExistingUUID(t *testing.T) {
	want := uuid.New()
	base := Base{ID: want}

	assert.NoError(t, base.BeforeCreate(nil))
	assert.Equal(t, want, base.ID)
}
