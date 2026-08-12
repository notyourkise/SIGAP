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
func SiteContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		siteID := c.Get("X-Site-ID")
		if siteID == "" {
			siteID = c.Query("site_id")
		}

		if siteID != "" {
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

			c.Locals("site_id", id.String())
		}

		return c.Next()
	}
}
