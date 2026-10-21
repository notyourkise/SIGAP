package middleware

import (
	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// SiteContext adalah middleware untuk membaca konteks site aktif dari request.
// Nilai diambil dari header X-Site-ID, dengan fallback ke query parameter
// ?site_id=. Bila nilai valid dan site tersebut ada di database, middleware
// menyimpan site_id ke c.Locals("site_id") untuk dipakai handler sebagai
// filter opsional. Bila TIDAK ada nilai yang dikirim, site_id dibiarkan kosong
// (tanpa filter) agar tidak merusak perilaku/test existing.
//
// HARDENING (isolasi multi-site): bila request sudah terautentikasi (middleware
// Protected() berjalan lebih dulu) dan pengguna BUKAN admin, pengguna hanya
// boleh memilih site tempat ia ditugaskan (tabel user_sites). Selain itu → 403.
func SiteContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		siteID := c.Get("X-Site-ID")
		if siteID == "" {
			siteID = c.Query("site_id")
		}

		if siteID == "" {
			return c.Next()
		}

		id, err := uuid.Parse(siteID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Format X-Site-ID tidak valid",
			})
		}

		// Verifikasi site benar-benar ada di database.
		var site models.Site
		if err := config.DB.First(&site, "id = ?", id).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Site tidak ditemukan",
			})
		}

		// Enforce keanggotaan site untuk user non-admin. Bila ditolak, respons
		// error sudah dikirim → hentikan rantai handler (jangan c.Next()).
		if enforceSiteMembership(c, id) {
			return nil
		}

		c.Locals("site_id", id.String())
		return c.Next()
	}
}

// enforceSiteMembership memastikan user non-admin yang terautentikasi memang
// ditugaskan ke site yang diminta. Mengembalikan true bila request DITOLAK dan
// respons error sudah dikirim ke client, sehingga pemanggil harus berhenti
// tanpa memanggil c.Next(). Bila konteks autentikasi tidak tersedia (mis. unit
// test tanpa middleware Protected()), pemeriksaan dilewati (mengembalikan
// false) agar tidak mengubah perilaku lama.
func enforceSiteMembership(c *fiber.Ctx, siteID uuid.UUID) bool {
	role, _ := c.Locals("role").(string)
	userIDStr, _ := c.Locals("user_id").(string)

	// Hanya berlaku pada request yang sudah melalui autentikasi.
	if role == "" || userIDStr == "" {
		return false
	}
	// Admin memiliki akses ke seluruh site.
	if role == "admin" {
		return false
	}

	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak valid",
		})
		return true
	}

	var count int64
	if err := config.DB.Table("user_sites").
		Where("site_id = ? AND user_id = ?", siteID, uid).
		Count(&count).Error; err != nil {
		_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memverifikasi akses site",
		})
		return true
	}
	if count == 0 {
		_ = c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: Anda tidak ditugaskan ke site ini",
		})
		return true
	}
	return false
}
