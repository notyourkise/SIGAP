package utils

import (
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExtractClientIP mengembalikan IP address klien (proxy-aware).
// Hanya membaca header proxy bila koneksi datang dari trusted proxy yang
// dikonfigurasi (app.SetTrustedProxies); jika tidak, c.IP() mengembalikan
// IP peer langsung sehingga header X-Forwarded-For tidak bisa dipalsukan klien.
func ExtractClientIP(c *fiber.Ctx) string {
	return c.IP()
}

// LogAudit menyimpan catatan audit log ke database secara asynchronous
func LogAudit(db *gorm.DB, c *fiber.Ctx, action string, module string, description string, userID *uuid.UUID, username string, role string) {
	if db == nil {
		return
	}
	ipAddress := ExtractClientIP(c)
	userAgent := c.Get("User-Agent")

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

	// Eksekusi penulisan di goroutine non-blocking
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

		if err := database.Create(&logData).Error; err != nil {
			Log.Errorf("❌ Gagal menyimpan AuditLog [%s]: %v", logData.Action, err)
		}
	}(audit, db)
}
