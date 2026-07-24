package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base struct untuk otomatisasi UUID, Timestamp, dan Soft Delete
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Hook GORM: Generate UUID secara otomatis sebelum data disimpan ke database
func (base *Base) BeforeCreate(tx *gorm.DB) (err error) {
	base.ID = uuid.New()
	return
}

type User struct {
	Base
	Username     string `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"` // Sembunyikan dari respon JSON API
	FullName     string `gorm:"type:varchar(100);not null" json:"full_name"`
	Email        string `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Role         string `gorm:"type:varchar(20);not null" json:"role"`
}

type Asset struct {
	Base
	Name             string    `gorm:"type:varchar(100);not null" json:"name"`
	SerialNumber     string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"serial_number"`
	IPAddress        string    `gorm:"type:varchar(45)" json:"ip_address"`
	LocationName     string    `gorm:"type:varchar(255)" json:"location_name"`
	Latitude         float64   `gorm:"type:decimal(10,8)" json:"latitude"`
	Longitude        float64   `gorm:"type:decimal(11,8)" json:"longitude"`
	Status           string    `gorm:"type:varchar(20);not null" json:"status"`
	Vendor           string    `gorm:"type:varchar(100)" json:"vendor"`
	InstallationDate time.Time `gorm:"type:date" json:"installation_date"`
}

type MaintenanceLog struct {
	Base
	AssetID       uuid.UUID  `gorm:"type:uuid;not null" json:"asset_id"`
	Asset         Asset      `gorm:"foreignKey:AssetID" json:"asset"`
	TechnicianID  uuid.UUID  `gorm:"type:uuid;not null" json:"technician_id"`
	User          User       `gorm:"foreignKey:TechnicianID" json:"technician"`
	ActionTaken   string     `gorm:"type:text;not null" json:"action_taken"`
	Notes         string     `gorm:"type:text" json:"notes"`
	Status        string     `gorm:"type:varchar(20);not null" json:"status"`
	ScheduledDate *time.Time `json:"scheduled_date"`
	CompletedAt   *time.Time `json:"completed_at"`
	ImageUrl      string     `gorm:"type:varchar(255)" json:"image_url,omitempty"`
}