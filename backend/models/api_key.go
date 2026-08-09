package models

import (
	"time"

	"github.com/google/uuid"
)

// Scope API key yang tersedia. API key hanya dapat mengakses endpoint yang
// mencantumkan scope-nya (Task 12.1, Sprint 29).
const (
	// ScopeCCTVRead mengizinkan akses baca data CCTV (GET /api/cctv).
	ScopeCCTVRead = "cctv:read"
	// ScopeMaintenanceRead mengizinkan akses baca log pemeliharaan.
	ScopeMaintenanceRead = "maintenance:read"
	// ScopeHealthRead mengizinkan akses cek kesehatan layanan (GET /api/health).
	ScopeHealthRead = "health:read"
)

// APIKey adalah kredensial integrasi untuk konsumen eksternal (sistem lain)
// yang memanggil API SIGAP tanpa sesi web. Hanya menyimpan hash bcrypt dari
// key — key mentah hanya ditampilkan sekali saat pembuatan.
type APIKey struct {
	Base
	// Name adalah label pengenal key (mis. "Integrasi Dashboard OIKN").
	Name string `gorm:"type:varchar(100);not null" json:"name"`
	// KeyHash adalah bcrypt hash dari key mentah. Tidak pernah dikembalikan.
	KeyHash string `gorm:"type:varchar(255);not null" json:"-"`
	// KeyPrefix adalah 8 karakter pertama key mentah — dipakai admin untuk
	// mengenali key tanpa menyimpan key penuh.
	KeyPrefix string `gorm:"type:varchar(16);not null;index" json:"key_prefix"`
	// Scopes adalah daftar izin (jsonb array): "cctv:read", "maintenance:read", "health:read".
	Scopes []string `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"scopes"`
	// ExpiresAt adalah waktu kedaluwarsa key (nil = tidak pernah kedaluwarsa).
	ExpiresAt *time.Time `json:"expires_at"`
	// CreatedBy menunjuk user admin pembuat key.
	CreatedBy uuid.UUID `gorm:"type:uuid;not null;index" json:"created_by"`
	User      User      `gorm:"foreignKey:CreatedBy" json:"user,omitempty"`
	// LastUsedAt mencatat waktu terakhir key berhasil digunakan (usage tracking).
	LastUsedAt *time.Time `json:"last_used_at"`
	// RevokedAt menandai key dicabut (hard revoke). Key yang dicabut ditolak.
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// UserID kembali disediakan agar API key dapat diambil dengan Preload("User").
func (APIKey) TableName() string {
	return "api_keys"
}
