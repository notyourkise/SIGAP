package models

import (
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedAdmin membuat akun Admin pertama kali jika database kosong
func SeedAdmin(db *gorm.DB) {
	var count int64
	db.Model(&User{}).Count(&count)

	if count == 0 {
		// Enkripsi password default admin
		password := "adminpassword"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("⚠️ Gagal mengenkripsi password admin default: ", err)
			return
		}

		admin := User{
			Username:     "admin",
			PasswordHash: string(hashedPassword),
			FullName:     "Admin SIGAP",
			Email:        "admin@sigap.local",
			Role:         "admin",
		}

		err = db.Create(&admin).Error
		if err != nil {
			log.Println("⚠️ Gagal membuat akun Admin default: ", err)
		} else {
			log.Println("🚀 Berhasil menyemai (seed) Admin default (username: admin, pass: adminpassword)")
		}
	}
}
