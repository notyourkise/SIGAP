package models

import (
	"github.com/google/uuid"
)

// AuditLog merekam setiap aktivitas penting di sistem SIGAP
type AuditLog struct {
	Base
	UserID      *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	User        *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Username    string     `gorm:"type:varchar(100);index;not null" json:"username"`
	UserRole    string     `gorm:"type:varchar(50);not null" json:"user_role"`
	Action      string     `gorm:"type:varchar(100);index;not null" json:"action"`
	Module      string     `gorm:"type:varchar(50);index;not null" json:"module"`
	Description string     `gorm:"type:text;not null" json:"description"`
	IPAddress   string     `gorm:"type:varchar(45);not null" json:"ip_address"`
	UserAgent   string     `gorm:"type:varchar(255)" json:"user_agent"`
}
