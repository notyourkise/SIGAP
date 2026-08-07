package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sync"
)

// aesKey menyimpan kunci AES-256 yang sudah diderivasi.
// Dihitung sekali saat pertama kali dibutuhkan, bukan setiap kali fungsi dipanggil.
var (
	aesKey     []byte
	aesKeyOnce sync.Once
)

// deriveKey menghasilkan kunci 32-byte (AES-256) menggunakan SHA-256 dari
// AES_SECRET_KEY. Secret dibaca dari Docker secret (file /run/secrets/aes_secret_key)
// dengan fallback ke environment variable AES_SECRET_KEY.
// Secret wajib disediakan (divalidasi di startup oleh RequireSecret).
// Hasil derivasi di-cache menggunakan sync.Once untuk efisiensi.
func deriveKey() []byte {
	aesKeyOnce.Do(func() {
		secret := GetSecret("AES_SECRET_KEY", "/run/secrets/aes_secret_key")
		hash := sha256.Sum256([]byte(secret))
		aesKey = hash[:]
	})
	return aesKey
}

// EncryptAES256 mengenkripsi string teks menjadi format hex terenkripsi (AES-256-GCM)
func EncryptAES256(plaintext string) (string, error) {
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// DecryptAES256 mendekripsi data hex terenkripsi kembali ke format string semula
func DecryptAES256(ciphertextHex string) (string, error) {
	key := deriveKey()
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
