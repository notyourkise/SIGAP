package handlers

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"os"
	"sigap-backend/aiassistant"
	"time"
)

// These are project requirements, not operational SOPs or live camera data.
var assistantSources = []aiassistant.Document{
	{ID: "KEBUTUHAN-01", Text: "Target rancangan SIGAP adalah membuat tiket dan mengirim pemberitahuan grup maksimal lima menit setelah CCTV benar-benar offline. Target ini belum dibuktikan pada perangkat nyata."},
	{ID: "KEBUTUHAN-02", Text: "Teknisi menerima pemberitahuan di grup Telegram tanpa menghubungkan akun Telegram ke SIGAP. Admin menugaskan teknisi melalui website."},
	{ID: "KEBUTUHAN-03", Text: "Pemulihan dikonfirmasi setelah CCTV teramati online stabil selama lima menit. Admin memverifikasi dan menutup insiden gangguan otomatis."},
	{ID: "KEBUTUHAN-04", Text: "Kegagalan pemantau tidak langsung membuktikan CCTV offline. Jika bukti tidak memadai, kondisi kamera belum dapat dipastikan."},
	{ID: "KEBUTUHAN-05", Text: "Monitoring aktif kamera GSM diperbolehkan dengan pembatasan durasi, timeout, percobaan ulang, dan jumlah pemeriksaan bersamaan. Penggunaan data aktual perlu diukur."},
}

func AskAIAssistant(c *fiber.Ctx) error {
	var req struct {
		Question string `json:"question"`
	}
	if c.BodyParser(&req) != nil || len([]rune(req.Question)) > 2000 || req.Question == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Pertanyaan wajib diisi, maksimal 2000 karakter"})
	}
	client := aiassistant.Client{BaseURL: os.Getenv("AI_GATEWAY_URL"), APIKey: os.Getenv("AI_GATEWAY_KEY"), Model: os.Getenv("AI_CHAT_MODEL"), EmbeddingModel: os.Getenv("AI_EMBEDDING_MODEL")}
	ctx, cancel := context.WithTimeout(c.UserContext(), 35*time.Second)
	defer cancel()
	result, err := client.Answer(ctx, req.Question, assistantSources)
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"status": "error", "message": "Asisten AI belum tersedia. Periksa konfigurasi gateway dan model pada server."})
	}
	return c.JSON(fiber.Map{"status": "success", "data": result})
}
