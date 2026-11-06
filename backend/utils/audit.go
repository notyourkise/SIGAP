package utils

import (
	"fmt"
	"sigap-backend/models"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	auditWriteAttempts = 3
	auditRetryDelay    = 100 * time.Millisecond
)

var (
	auditHookMu sync.RWMutex
	auditHook   func(models.AuditLog)
)

// SetAuditHookForTest mendaftarkan hook pengujian untuk menginspeksi AuditLog yang dibangun.
// Mengembalikan fungsi cleanup untuk memulihkan state semula.
func SetAuditHookForTest(fn func(models.AuditLog)) func() {
	auditHookMu.Lock()
	auditHook = fn
	auditHookMu.Unlock()
	return func() {
		auditHookMu.Lock()
		auditHook = nil
		auditHookMu.Unlock()
	}
}

// ExtractClientIP mengembalikan IP address klien (proxy-aware).
// Hanya membaca header X-Forwarded-For bila koneksi datang dari trusted proxy
// yang dikonfigurasi via TRUSTED_PROXY_IPS (config.NewFiberConfig); jika tidak,
// c.IP() mengembalikan IP peer langsung sehingga header X-Forwarded-For tidak
// bisa dipalsukan klien.
func ExtractClientIP(c *fiber.Ctx) string {
	return c.IP()
}

// LogAudit menyimpan catatan audit log ke database secara asynchronous
func LogAudit(db *gorm.DB, c *fiber.Ctx, action string, module string, description string, userID *uuid.UUID, username string, role string) {
	if db == nil {
		return
	}
	var ipAddress, userAgent string
	if c != nil {
		ipAddress = ExtractClientIP(c)
		userAgent = c.Get("User-Agent")

		if len(userAgent) > 255 {
			userAgent = userAgent[:255]
		}

		// Auto-extract dari c.Locals jika tidak dipassing secara eksplisit
		if userID == nil {
			if uidStr, ok := c.Locals("user_id").(string); ok && uidStr != "" {
				if parsedID, err := uuid.Parse(uidStr); err == nil {
					userID = &parsedID
				}
			}
		}

		if role == "" {
			if r, ok := c.Locals("role").(string); ok {
				role = r
			}
		}
	}

	audit := models.AuditLog{
		UserID:      userID,
		Username:    username,
		UserRole:    role,
		Action:      action,
		Module:      module,
		Description: description,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}

	auditHookMu.RLock()
	hook := auditHook
	auditHookMu.RUnlock()
	if hook != nil {
		hook(audit)
	}

	// Eksekusi penulisan di goroutine non-blocking. Retry terbatas menangani
	// gangguan koneksi singkat tanpa membuat request operasional ikut menunggu.
	go func(logData models.AuditLog, database *gorm.DB) {
		// Lookup username dari DB jika belum terisi namun userID ada
		if logData.Username == "" && logData.UserID != nil {
			var u models.User
			if err := database.Select("username", "role").First(&u, "id = ?", *logData.UserID).Error; err == nil {
				logData.Username = u.Username
				if logData.UserRole == "" {
					logData.UserRole = u.Role
				}
			}
		}
		if logData.Username == "" {
			logData.Username = "system"
		}
		if logData.UserRole == "" {
			logData.UserRole = "guest"
		}

		if err := persistAuditLog(database, &logData); err != nil {
			Log.Errorf("❌ Gagal menyimpan AuditLog [%s] setelah %d percobaan: %v", logData.Action, auditWriteAttempts, err)
			return
		}

		// Audit log memuat IP, user-agent, dan deskripsi aktivitas. Siarkan
		// hanya ke koneksi admin; non-admin tidak boleh menerima event ini.
		BroadcastToRole("admin", "AUDIT_LOG_ADDED", logData)
	}(audit, db)
}

func persistAuditLog(database *gorm.DB, logData *models.AuditLog) error {
	var lastErr error
	for attempt := 1; attempt <= auditWriteAttempts; attempt++ {
		if err := database.Create(logData).Error; err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt < auditWriteAttempts {
			time.Sleep(auditRetryDelay)
		}
	}

	return fmt.Errorf("audit log write failed: %w", lastErr)
}
