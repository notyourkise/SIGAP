package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateAPIKeyRequest adalah body pembuatan API key (Admin saja).
type CreateAPIKeyRequest struct {
	Name      string   `json:"name" validate:"required,min=2,max=100"`
	Scopes    []string `json:"scopes" validate:"required"`
	ExpiresIn int      `json:"expires_in_days" validate:"omitempty,gte=1"` // 0 = tidak pernah
}

// generateAPIKey menghasilkan key mentah acak (32 byte hex = 64 karakter) dan
// prefix-nya (8 karakter pertama).
func generateAPIKey() (string, string) {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	key := hex.EncodeToString(buf)
	return key, key[:8]
}

// validAPIScope memeriksa apakah daftar scope valid (terdaftar).
func validAPIScope(scope string) bool {
	switch scope {
	case models.ScopeCCTVRead, models.ScopeMaintenanceRead, models.ScopeHealthRead:
		return true
	}
	return false
}

// CreateAPIKey membuat API key baru dan mengembalikan key mentah (hanya 1×).
// @Summary      Buat API Key
// @Description  Membuat API key baru untuk integrasi eksternal. Key mentah hanya ditampilkan sekali pada respons ini; hash bcrypt disimpan di database. Dapat diakses hanya oleh Admin.
// @Tags         API Key
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateAPIKeyRequest true "Data API Key Baru"
// @Success      201 {object} map[string]interface{} "API key berhasil dibuat (key ditampilkan sekali)"
// @Failure      400 {object} map[string]interface{} "Validasi gagal atau scope tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal membuat API key"
// @Router       /api-keys [post]
func CreateAPIKey(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak ditemukan",
		})
	}
	createdBy, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak valid",
		})
	}

	var req CreateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}
	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Validasi scope.
	for _, s := range req.Scopes {
		if !validAPIScope(s) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Scope tidak valid. Gunakan cctv:read, maintenance:read, atau health:read",
			})
		}
	}

	rawKey, prefix := generateAPIKey()
	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghasilkan hash API key",
		})
	}

	item := models.APIKey{
		Name:      req.Name,
		KeyHash:   string(hash),
		KeyPrefix: prefix,
		Scopes:    req.Scopes,
		CreatedBy: createdBy,
	}
	if req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresIn) * 24 * time.Hour)
		item.ExpiresAt = &t
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan API key",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_API_KEY", "API_KEY",
		"Membuat API key: "+item.Name, nil, "", "")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"id":        item.ID,
			"name":      item.Name,
			"key_prefix": item.KeyPrefix,
			"scopes":    item.Scopes,
			"expires_at": item.ExpiresAt,
			"api_key":   rawKey, // hanya tampil sekali
			"message":   "Simpan API key ini sekarang; tidak akan ditampilkan lagi.",
		},
	})
}

// GetAllAPIKeys mengambil daftar API key terpaginasi (Admin & Teknisi yang
// login) beserta info penggunaan (last_used_at, revoked, expires).
// @Summary      Ambil Semua API Key
// @Description  Mendapatkan daftar API key terpaginasi. Hash tidak dikembalikan; hanya prefix dan metadata.
// @Tags         API Key
// @Produce      json
// @Security     BearerAuth
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        search query string false "Pencarian berdasarkan nama"
// @Success      200 {object} map[string]interface{} "Daftar API key terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar API key"
// @Router       /api-keys [get]
func GetAllAPIKeys(c *fiber.Ctx) error {
	var items []models.APIKey
	query := config.DB.Model(&models.APIKey{})

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	allowedSort := []string{"id", "name", "created_at", "last_used_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total API key",
		})
	}

	offset := (page - 1) * limit
	if err := query.Preload("User").Offset(offset).Limit(limit).Order(sort).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar API key",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       items,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// RevokeAPIKey mencabut API key (soft revoke via revoked_at) — Admin saja.
// @Summary      Cabut API Key
// @Description  Mencabut API key sehingga tidak dapat digunakan lagi. Key tetap tersimpan untuk audit (revoked_at diisi).
// @Tags         API Key
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "API Key UUID"
// @Success      200 {object} map[string]interface{} "API key berhasil dicabut"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "API key tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal mencabut API key"
// @Router       /api-keys/{id}/revoke [post]
func RevokeAPIKey(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID API key tidak valid",
		})
	}

	var item models.APIKey
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "API key tidak ditemukan",
		})
	}

	now := time.Now()
	if err := config.DB.Model(&models.APIKey{}).Where("id = ?", id).
		Update("revoked_at", now).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mencabut API key",
		})
	}

	utils.LogAudit(config.DB, c, "REVOKE_API_KEY", "API_KEY",
		"Mencabut API key: "+item.Name, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "API key berhasil dicabut",
	})
}
