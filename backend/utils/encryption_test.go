package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptDecryptAES256_RoundTrip(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")
	plaintext := "SIGAP-CCTV-2026-HEAVY-KEY"

	ciphertextHex, err := EncryptAES256(plaintext)
	assert.NoError(t, err)
	assert.NotEmpty(t, ciphertextHex)

	// Hasil enkripsi harus berbeda dari plaintext (bukan plaintext mentah)
	assert.NotEqual(t, plaintext, ciphertextHex)

	decrypted, err := DecryptAES256(ciphertextHex)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptAES256_TwoCallsProduceDifferentCiphertexts(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")

	// GCM memakai nonce acak → ciphertext dua pemanggilan harus berbeda
	first, err := EncryptAES256("data-rahasia")
	assert.NoError(t, err)
	second, err := EncryptAES256("data-rahasia")
	assert.NoError(t, err)
	assert.NotEqual(t, first, second)

	// Namun keduanya harus bisa didekripsi ke plaintext yang sama
	dec1, err := DecryptAES256(first)
	assert.NoError(t, err)
	dec2, err := DecryptAES256(second)
	assert.NoError(t, err)
	assert.Equal(t, dec1, dec2)
}

func TestDecryptAES256_InvalidHex(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")

	_, err := DecryptAES256("bukan-hex-valid-@@@")
	assert.Error(t, err)
}

func TestDecryptAES256_EmptyInput(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")

	_, err := DecryptAES256("")
	assert.Error(t, err)
}
