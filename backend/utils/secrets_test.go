package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSecretFromFile memastikan bila file secret tersedia, isinya
// (dengan whitespace ujung yang di-trim) diprioritaskan di atas env var.
func TestSecretFromFile(t *testing.T) {
	t.Setenv("MY_SECRET", "nilai-dari-env")

	dir := t.TempDir()
	secretFile := filepath.Join(dir, "my_secret")
	err := os.WriteFile(secretFile, []byte("nilai-dari-file\n"), 0600)
	assert.NoError(t, err)

	got := GetSecret("MY_SECRET", secretFile)
	assert.Equal(t, "nilai-dari-file", got, "nilai file harus menang atas env var")
}

// TestSecretFallsBackToEnv memastikan bila file secret tidak ada,
// nilai diambil dari environment variable (fallback).
func TestSecretFallsBackToEnv(t *testing.T) {
	t.Setenv("MY_SECRET", "nilai-dari-env")

	// Path file yang dijamin tidak ada di temp dir kosong.
	secretFile := filepath.Join(t.TempDir(), "tidak_ada")

	got := GetSecret("MY_SECRET", secretFile)
	assert.Equal(t, "nilai-dari-env", got, "harus fallback ke env var bila file tidak ada")
}

// TestSecretEmptyWhenBothMissing memastikan bila file maupun env tidak ada,
// fungsi mengembalikan string kosong (bukan nilai hardcoded).
func TestSecretEmptyWhenBothMissing(t *testing.T) {
	t.Setenv("MY_SECRET", "")

	secretFile := filepath.Join(t.TempDir(), "tidak_ada")

	got := GetSecret("MY_SECRET", secretFile)
	assert.Equal(t, "", got, "harus kosong bila file & env tidak tersedia")
}

// TestSecretTrimsWhitespace memastikan whitespace (spasi & newline)
// di sekitar isi file secret dibuang sebelum dikembalikan.
func TestSecretTrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	secretFile := filepath.Join(dir, "my_secret")
	err := os.WriteFile(secretFile, []byte("  rahasia-sigap  \n"), 0600)
	assert.NoError(t, err)

	got := GetSecret("MY_SECRET", secretFile)
	assert.Equal(t, "rahasia-sigap", got)
}
