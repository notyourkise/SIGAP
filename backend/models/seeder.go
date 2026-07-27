package models

import (
	"log"
	"time"

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

// SeedAssets membuat beberapa aset CCTV default jika database kosong
func SeedAssets(db *gorm.DB) {
	var count int64
	db.Model(&Asset{}).Count(&count)

	if count == 0 {
		assets := []Asset{
			{
				Name:             "CCTV Bundaran HI Utama",
				SerialNumber:     "CCTV-BND-HI-001",
				IPAddress:        "192.168.10.11",
				LocationName:     "Kawasan Bundaran HI, Jakarta Pusat",
				Latitude:         -6.1949,
				Longitude:        106.8231,
				Status:           "aktif",
				Vendor:           "Hikvision",
				InstallationDate: time.Now().AddDate(-1, 0, 0), // 1 tahun lalu
			},
			{
				Name:             "CCTV Pintu Masuk Monas Barat",
				SerialNumber:     "CCTV-MNS-WRT-002",
				IPAddress:        "192.168.10.12",
				LocationName:     "Monumen Nasional Pintu Barat, Jakarta Pusat",
				Latitude:         -6.1754,
				Longitude:        106.8222,
				Status:           "aktif",
				Vendor:           "Dahua",
				InstallationDate: time.Now().AddDate(0, -6, 0), // 6 bulan lalu
			},
			{
				Name:             "CCTV Koridor Istana Negara",
				SerialNumber:     "CCTV-IST-NGR-003",
				IPAddress:        "192.168.20.15",
				LocationName:     "Istana Negara Koridor A, Jakarta Tengah",
				Latitude:         -6.1698,
				Longitude:        106.8218,
				Status:           "pemeliharaan",
				Vendor:           "Bosch",
				InstallationDate: time.Now().AddDate(0, -2, 0), // 2 bulan lalu
			},
			{
				Name:             "CCTV Tol Simpang Susun Semanggi",
				SerialNumber:     "CCTV-SMG-TOL-004",
				IPAddress:        "192.168.30.55",
				LocationName:     "Simpang Susun Semanggi, Jakarta Selatan",
				Latitude:         -6.2201,
				Longitude:        106.8116,
				Status:           "rusak",
				Vendor:           "Sony",
				InstallationDate: time.Now().AddDate(-2, 0, 0), // 2 tahun lalu
			},
		}

		for _, asset := range assets {
			err := db.Create(&asset).Error
			if err != nil {
				log.Printf("⚠️ Gagal menyemai aset %s: %v\n", asset.Name, err)
			}
		}
		log.Println("🚀 Berhasil menyemai (seed) 4 Aset CCTV default ke database")
	}
}
