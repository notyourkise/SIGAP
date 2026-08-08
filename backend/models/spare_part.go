package models

import (
	"github.com/google/uuid"
)

// Kategori default suku cadang — dipakai frontend untuk filter dropdown dan
// menjaga konsistensi penamaan kategori inventaris.
const (
	SparePartCategoryKabel       = "Kabel"
	SparePartCategoryKonektor    = "Konektor"
	SparePartCategoryPowerSupply = "Power Supply"
	SparePartCategoryLensa       = "Lensa"
	SparePartCategoryStorage     = "Storage"
	SparePartCategoryLainnya     = "Lainnya"
)

// SparePart adalah master data suku cadang / komponen (kabel, konektor,
// power supply, dll.) yang digunakan saat perbaikan CCTV. Stok dikurangi
// otomatis ketika teknisi mencatat penggunaan suku cadang pada maintenance log.
type SparePart struct {
	Base
	// Nama adalah nama suku cadang / komponen (mis. "Kabel UTP Cat6 5m").
	Nama string `gorm:"type:varchar(100);not null" json:"nama"`
	// SKU adalah kode unik inventaris (Stock Keeping Unit).
	SKU string `gorm:"type:varchar(50);uniqueIndex" json:"sku"`
	// Kategori mengelompokkan jenis komponen (Kabel, Konektor, Power Supply, dll.).
	Kategori string `gorm:"type:varchar(50)" json:"kategori"`
	// Stok adalah jumlah unit yang tersedia saat ini.
	Stok int `gorm:"type:integer;not null;default:0" json:"stok"`
	// MinimumOrder adalah ambang batas stok minimum untuk peringatan (STOCK_LOW).
	MinimumOrder int `gorm:"type:integer;not null;default:5" json:"minimum_order"`
	// Satuan adalah unit penghitung stok (pcs, meter, rol, dll).
	Satuan string `gorm:"type:varchar(20);not null;default:'pcs'" json:"satuan"`
	// HargaSatuan adalah harga per satuan untuk referensi biaya.
	HargaSatuan float64 `gorm:"type:decimal(12,2);default:0" json:"harga_satuan"`
	// Keterangan berisi catatan tambahan tentang suku cadang.
	Keterangan string `gorm:"type:text" json:"keterangan"`
}

// SparePartUsage mencatat penggunaan suku cadang pada sebuah maintenance log.
// Saat record ini dibuat, stok SparePart terkait dikurangi otomatis (dalam
// transaksi database) sejumlah Jumlah.
type SparePartUsage struct {
	Base
	// MaintenanceLogID menunjuk ke log pemeliharaan tempat suku cadang dipakai.
	MaintenanceLogID uuid.UUID      `gorm:"type:uuid;not null;index" json:"maintenance_log_id"`
	MaintenanceLog   MaintenanceLog `gorm:"foreignKey:MaintenanceLogID" json:"maintenance_log,omitempty"`
	// SparePartID menunjuk ke master suku cadang yang digunakan.
	SparePartID uuid.UUID `gorm:"type:uuid;not null;index" json:"spare_part_id"`
	SparePart   SparePart `gorm:"foreignKey:SparePartID" json:"spare_part,omitempty"`
	// Jumlah adalah banyaknya unit yang dipakai (harus > 0 dan <= stok tersedia).
	Jumlah int `gorm:"type:integer;not null" json:"jumlah"`
	// Catatan berisi keterangan penggunaan (opsional).
	Catatan string `gorm:"type:text" json:"catatan"`
}
