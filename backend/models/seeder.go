package models

import (
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedAdmin membuat akun Admin default HANYA jika belum ada.
// Tidak ada fallback password hardcoded; ADMIN_DEFAULT_PASSWORD wajib diset saat pembuatan akun pertama.
// Password admin yang sudah tersimpan di database TIDAK di-reset ulang pada setiap startup.
func SeedAdmin(db *gorm.DB) {
	var admin User
	errDB := db.Where("username = ?", "admin").First(&admin).Error
	if errDB == nil {
		// Akun admin sudah ada — biarkan password yang ada tetap berlaku.
		log.Println("✅ Akun Admin default sudah ada (username: admin), password tidak diubah.")
		return
	}

	password := os.Getenv("ADMIN_DEFAULT_PASSWORD")
	if password == "" {
		log.Fatalf("❌ Environment variable ADMIN_DEFAULT_PASSWORD wajib diisi untuk membuat akun admin pertama kali. Server dihentikan.")
	}
	if len(password) < 8 {
		log.Fatalf("❌ ADMIN_DEFAULT_PASSWORD terlalu pendek (minimal 8 karakter). Server dihentikan.")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ Gagal mengenkripsi password admin default: %v", err)
	}

	// Akun Admin belum ada, buat baru
	admin = User{
		Username:           "admin",
		PasswordHash:       string(hashedPassword),
		FullName:           "Admin SIGAP",
		Email:              "admin@sigap.local",
		Role:               "admin",
		MustChangePassword: true, // paksa ganti password pada login pertama
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("❌ Gagal membuat akun Admin default: %v", err)
	}
	log.Println("🚀 Akun Admin default berhasil dibuat (username: admin)")
}
