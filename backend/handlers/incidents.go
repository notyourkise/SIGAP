package handlers

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sigap-backend/config"
	"sigap-backend/incidents"
	"sigap-backend/utils"
)

func IncidentMonitoringEnabled() bool { return os.Getenv("CCTV_INCIDENTS_ENABLED") == "true" }

func incidentSite(c *fiber.Ctx) (uuid.UUID, bool) {
	value, _ := c.Locals("site_id").(string)
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		_ = c.Status(400).JSON(fiber.Map{"status": "error", "message": "Pilih site untuk mengelola insiden CCTV"})
		return uuid.Nil, false
	}
	return id, true
}

func GetIncidents(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	var items []incidents.Incident
	if err := config.DB.Preload("Members").Where("site_id = ?", site).Order("opened_at DESC").Limit(100).Find(&items).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Tidak dapat memuat tiket insiden"})
	}
	var jobs []incidents.Outbox
	if err := config.DB.Where("site_id = ?", site).Order("created_at DESC").Limit(50).Find(&jobs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Tidak dapat memuat status pemberitahuan"})
	}
	return c.JSON(fiber.Map{"status": "success", "data": fiber.Map{"incidents": items, "notifications": jobs, "monitoring_enabled": IncidentMonitoringEnabled()}})
}

func AssignIncident(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID insiden tidak valid"})
	}
	var req struct {
		TechnicianID string `json:"technician_id"`
	}
	if c.BodyParser(&req) != nil {
		return c.SendStatus(400)
	}
	tech, err := uuid.Parse(req.TechnicianID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID teknisi tidak valid"})
	}
	if err := (incidents.Service{DB: config.DB}).Assign(c.UserContext(), id, site, tech, time.Now().UTC()); err != nil {
		return incidentActionError(c, err)
	}
	return c.JSON(fiber.Map{"status": "success", "message": "Teknisi ditugaskan; tugas tersedia di menu pemeliharaan"})
}

func ResolveIncident(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.SendStatus(400)
	}
	user, _ := c.Locals("user_id").(string)
	admin, err := uuid.Parse(user)
	if err != nil {
		return c.SendStatus(401)
	}
	var req struct {
		Note string `json:"note"`
	}
	if c.BodyParser(&req) != nil || len(req.Note) > 2000 {
		return c.SendStatus(400)
	}
	if err := (incidents.Service{DB: config.DB}).Resolve(c.UserContext(), id, site, admin, req.Note, time.Now().UTC()); err != nil {
		return incidentActionError(c, err)
	}
	return c.JSON(fiber.Map{"status": "success", "message": "Insiden telah diverifikasi dan ditutup"})
}

func incidentActionError(c *fiber.Ctx, err error) error {
	if err == gorm.ErrRecordNotFound {
		return c.SendStatus(404)
	}
	// Log details internally; never echo database connection errors to a client.
	utils.Log.Warnf("Tindakan insiden ditolak: %v", err)
	return c.Status(409).JSON(fiber.Map{"status": "error", "message": "Tindakan belum dapat dilakukan. Pastikan site, teknisi, dan bukti pemulihan masih valid."})
}

func RunIncidentNotifications(db *gorm.DB) {
	if !IncidentMonitoringEnabled() {
		return
	}
	ctx := context.Background()
	if err := (incidents.Service{DB: db}).ResetObservations(ctx); err != nil {
		utils.Log.Errorf("Gagal membatalkan observasi lama: %v", err)
		return
	}
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chats := map[string]string{}
	if token == "" || json.Unmarshal([]byte(os.Getenv("TELEGRAM_SITE_CHATS_JSON")), &chats) != nil || len(chats) == 0 {
		utils.Log.Warn("Antrean insiden aktif; pengiriman Telegram belum dikonfigurasi. Pesan tetap tersimpan.")
		return
	}
	sender := incidents.TelegramSender{Token: token}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for now := range ticker.C {
		if _, err := incidents.DeliverOne(ctx, db, sender, chats, now.UTC()); err != nil {
			utils.Log.Errorf("Antrean Telegram gagal diproses: %v", err)
		}
	}
}
