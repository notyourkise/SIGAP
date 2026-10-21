package middleware

import (
	"errors"
	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// siteCookieName adalah cookie preferensi site yang ditulis frontend. Header
// X-Site-ID tetap menjadi sumber utama; cookie dipakai sebagai fallback untuk
// request yang tidak membawa header (mis. API langsung dari browser).
const siteCookieName = "sigap_site_id"

// SiteContext adalah middleware untuk membaca konteks site aktif dari request.
// Nilai diambil dari header X-Site-ID, lalu cookie sigap_site_id, lalu query
// parameter ?site_id=. Bila nilai valid dan site tersebut ada di database,
// middleware menyimpan site_id ke c.Locals("site_id") untuk dipakai handler
// sebagai filter opsional.
//
// HARDENING (isolasi multi-site):
//   - Bila request sudah terautentikasi (middleware Protected() berjalan lebih
//     dulu) dan pengguna BUKAN admin, pengguna hanya boleh memilih site tempat
//     ia ditugaskan (tabel user_sites). Selain itu -> 403.
//   - Bila non-admin tidak mengirim site sama sekali (mis. cookie sigap_site_id
//     belum ada), middleware memilih site penugasan default pengguna. Bila
//     pengguna belum punya penugasan sama sekali -> 403 (fail closed), bukan
//     membuka seluruh data lintas site.
func SiteContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		siteID := c.Get("X-Site-ID")
		if siteID == "" {
			siteID = c.Cookies(siteCookieName)
		}
		if siteID == "" {
			siteID = c.Query("site_id")
		}

		if siteID == "" {
			// Admin tanpa site aktif sengaja dibiarkan kosong agar tetap dapat
			// melihat seluruh kawasan. Non-admin wajib mendapat scope default.
			if role, _ := c.Locals("role").(string); role != "" && role != "admin" {
				resolved, ok := defaultSiteForUser(c)
				if !ok {
					return nil
				}
				c.Locals("site_id", resolved)
			}
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
		// error sudah dikirim -> hentikan rantai handler (jangan c.Next()).
		if enforceSiteMembership(c, id) {
			return nil
		}

		c.Locals("site_id", id.String())
		return c.Next()
	}
}

// defaultSiteForUser memilih site default untuk user non-admin yang tidak
// mengirim konteks site. Prioritas: site penugasan yang bertanda default,
// lalu site penugasan pertama secara alfabetis. Bila user belum punya
// penugasan sama sekali, akses ditolak (fail closed).
//
// Mengembalikan false bila respons error sudah dikirim ke client.
func defaultSiteForUser(c *fiber.Ctx) (string, bool) {
	userIDStr, _ := c.Locals("user_id").(string)
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak valid",
		})
		return "", false
	}

	var site models.Site
	err = config.DB.
		Joins("JOIN user_sites ON user_sites.site_id = sites.id").
		Where("user_sites.user_id = ?", uid).
		Order("sites.is_default DESC, sites.nama ASC").
		First(&site).Error
	if err == nil {
		return site.ID.String(), true
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menentukan site pengguna",
		})
		return "", false
	}

	// Tidak ada penugasan site: jangan fallback ke seluruh data atau ke site
	// yang bukan milik pengguna. Tolak sampai admin menetapkan penugasan.
	_ = c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"status":  "error",
		"message": "Anda belum ditugaskan ke site mana pun",
	})
	return "", false
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
