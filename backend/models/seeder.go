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

// SeedChecklistTemplates mengisi tabel checklist_templates dengan 9 item
// template default HANYA jika tabel masih kosong (Task 13.12). Data default
// disediakan agar daftar checklist pengecekan lapangan siap dipakai begitu
// server pertama kali dijalankan. Fungsi idempotent — bila sudah ada data,
// tidak melakukan apa-apa agar kustomisasi admin tidak tertimpa.
func SeedChecklistTemplates(db *gorm.DB) {
	var existing ChecklistTemplate
	if err := db.First(&existing).Error; err == nil {
		// Template sudah terisi — biarkan kustomisasi admin yang ada tetap berlaku.
		log.Println("✅ Template checklist pengecekan sudah ada, kustomisasi admin tidak diubah.")
		return
	}

	defaults := []ChecklistTemplate{
		{NamaItem: "Pengecekan jaringan", Deskripsi: "Pastikan koneksi jaringan kamera ke NVR normal.", Urutan: 1, Wajib: true, Aktif: true},
		{NamaItem: "Pengecekan pasokan listrik", Deskripsi: "Periksa adaptor/PoE/panel surya; pastikan tegangan stabil.", Urutan: 2, Wajib: true, Aktif: true},
		{NamaItem: "Pembersihan lensa", Deskripsi: "Bersihkan lensa dari debu/kotoran/sarang laba-laba.", Urutan: 3, Wajib: true, Aktif: true},
		{NamaItem: "Pemeriksaan fisik kamera", Deskripsi: "Periksa housing, bracket, segel kedap air.", Urutan: 4, Wajib: true, Aktif: true},
		{NamaItem: "Pengecekan kualitas gambar", Deskripsi: "Pastikan fokus, kecerahan, warna gambar baik.", Urutan: 5, Wajib: true, Aktif: true},
		{NamaItem: "Pengujian perekaman", Deskripsi: "Pastikan rekaman tersimpan normal di NVR.", Urutan: 6, Wajib: true, Aktif: true},
		{NamaItem: "Pengecekan akses jarak jauh", Deskripsi: "Pastikan kamera dapat diakses remote view.", Urutan: 7, Wajib: true, Aktif: true},
		{NamaItem: "Pemeriksaan kabel & konektor", Deskripsi: "Periksa kabel, konektor, jalur instalasi.", Urutan: 8, Wajib: true, Aktif: true},
		{NamaItem: "Pengecekan lingkungan sekitar", Deskripsi: "Pastikan area sekitar kamera bersih, tidak ada halangan.", Urutan: 9, Wajib: true, Aktif: true},
	}

	if err := db.Create(&defaults).Error; err != nil {
		log.Fatalf("❌ Gagal membuat template checklist default: %v", err)
	}
	log.Println("🚀 Template checklist pengecekan default berhasil dibuat (9 item)")
}

// EnsureDefaultSite memastikan site default "OIKN" selalu ada saat startup
// (Task 12.2, Sprint 30). Fungsi idempotent: bila site dengan kode OIKN sudah
// ada, tidak melakukan apa-apa. Guard "tabel belum ada" menjaga fungsi tetap
// aman bila dijalankan di luar urutan migrasi (mis. pada test).
func EnsureDefaultSite(db *gorm.DB) {
	var site Site
	if err := db.Where("kode = ?", "OIKN").First(&site).Error; err == nil {
		// Site default sudah ada — biarkan tetap.
		return
	}

	// Tabel belum ada (atau belum ada data OIKN). Coba buat idempoten.
	newSite := Site{
		Nama:      "Kawasan OIKN",
		Kode:      "OIKN",
		Alamat:    "Ibu Kota Nusantara, Kalimantan Timur",
		IsDefault: true,
	}
	if err := db.Create(&newSite).Error; err != nil {
		log.Printf("⚠️ Gagal memastikan site default OIKN: %v", err)
		return
	}
	log.Println("🚀 Site default OIKN berhasil dipastikan ada")
}
