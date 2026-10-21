package handlers

import (
	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetPushPublicKey memberikan public key VAPID yang sama dengan yang dipakai
// backend saat mengirim push. Dengan demikian frontend tidak lagi bergantung
// pada nilai NEXT_PUBLIC_* yang dapat berbeda saat build/deploy.
func GetPushPublicKey(c *fiber.Ctx) error {
	if err := utils.ValidateVAPIDConfiguration(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "error",
			"message": "Notifikasi push belum dikonfigurasi oleh administrator sistem",
		})
	}
	key, err := utils.VAPIDPublicKey()
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "error",
			"message": "Notifikasi push belum dikonfigurasi oleh administrator sistem",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"public_key": key,
		},
	})
}

// SubscribePushRequest adalah payload untuk mendaftarkan langganan Web Push.
type SubscribePushRequest struct {
	Endpoint  string `json:"endpoint" validate:"required,url"`
	KeyP256DH string `json:"key_p256dh" validate:"required"`
	KeyAuth   string `json:"key_auth" validate:"required"`
}

// SubscribePush menyimpan langganan Web Push milik user yang sedang login
// (Task 8.3). Jika endpoint yang sama sudah terdaftar untuk user, data kunci
// diperbarui (idempotent) alih-alih membuat baris duplikat.
// @Summary      Daftarkan Langganan Push
// @Description  Menyimpan subscription Web Push (browser) untuk user yang login agar dapat menerima notifikasi push.
// @Tags         Push Notification
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body SubscribePushRequest true "Data subscription Web Push"
// @Success      200 {object} map[string]interface{} "Subscription tersimpan"
// @Failure      400 {object} map[string]interface{} "Payload tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Router       /push/subscribe [post]
func SubscribePush(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	var req SubscribePushRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}
	if req.Endpoint == "" || req.KeyP256DH == "" || req.KeyAuth == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "endpoint, key_p256dh, dan key_auth wajib diisi",
		})
	}

	// Cari subscription dengan endpoint yang sama (termasuk yang soft-deleted)
	// agar subscribe bersifat idempotent — tidak membuat duplikat.
	var existing models.PushSubscription
	err = config.DB.Unscoped().
		Where("user_id = ? AND endpoint = ?", userID, req.Endpoint).
		First(&existing).Error

	if err == nil {
		// Endpoint sudah ada: perbarui kunci & pulihkan bila pernah dihapus.
		updates := map[string]interface{}{
			"key_p256dh": req.KeyP256DH,
			"key_auth":   req.KeyAuth,
			"deleted_at": nil,
		}
		if err := config.DB.Unscoped().Model(&existing).Updates(updates).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal memperbarui langganan push",
			})
		}
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Langganan push diperbarui",
		})
	}

	sub := models.PushSubscription{
		UserID:    userID,
		Endpoint:  req.Endpoint,
		KeyP256DH: req.KeyP256DH,
		KeyAuth:   req.KeyAuth,
	}
	if err := config.DB.Create(&sub).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan langganan push",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Langganan push berhasil disimpan",
	})
}

// UnsubscribePush menghapus langganan Web Push milik user berdasarkan endpoint
// (Task 8.3). Endpoint dikirim pada body; operasi idempotent.
// @Summary      Berhenti Berlangganan Push
// @Description  Menghapus subscription Web Push user berdasarkan endpoint (opt-out).
// @Tags         Push Notification
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body map[string]string true "Body berisi endpoint"
// @Success      200 {object} map[string]interface{} "Subscription dihapus"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Router       /push/unsubscribe [delete]
func UnsubscribePush(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}
	if req.Endpoint == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "endpoint wajib diisi",
		})
	}

	// Soft delete baris milik user dengan endpoint tersebut.
	if err := config.DB.Where("user_id = ? AND endpoint = ?", userID, req.Endpoint).
		Delete(&models.PushSubscription{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus langganan push",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Langganan push dihapus",
	})
}

// currentUserID mengekstrak UUID user dari Locals (dari middleware Protected).
func currentUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Sesi tidak valid")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Identitas pengguna tidak valid")
	}
	return userID, nil
}
