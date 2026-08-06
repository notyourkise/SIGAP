package models

import "time"

// DailyDigest menyimpan ringkasan operasional satu kali per tanggal WITA.
// DigestDate menjadi kunci idempotensi generator agar restart server tidak
// membuat ringkasan dan notifikasi ganda pada tanggal yang sama.
type DailyDigest struct {
	Base
	DigestDate      time.Time `gorm:"type:date;not null;uniqueIndex" json:"digest_date"`
	NewTasks        int       `gorm:"not null;default:0" json:"new_tasks"`
	CompletedTasks  int       `gorm:"not null;default:0" json:"completed_tasks"`
	FailedTasks     int       `gorm:"not null;default:0" json:"failed_tasks"`
	OverdueTasks    int       `gorm:"not null;default:0" json:"overdue_tasks"`
	FieldChecks     int       `gorm:"not null;default:0" json:"field_checks"`
	AttentionChecks int       `gorm:"not null;default:0" json:"attention_checks"`
	StatusChanges   int       `gorm:"not null;default:0" json:"status_changes"`
	Summary         string    `gorm:"type:text;not null" json:"summary"`
}
