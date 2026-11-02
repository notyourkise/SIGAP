package handlers

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"os"
	"sigap-backend/aiassistant"
	"time"
)

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
	docs, err := aiassistant.Corpus()
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"status": "error", "message": "Fontes de conhecimento indisponíveis"})
	}
	result, err := client.Answer(ctx, req.Question, docs)
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"status": "error", "message": "Asisten AI belum tersedia. Periksa konfigurasi gateway dan model pada server."})
	}
	return c.JSON(fiber.Map{"status": "success", "data": result})
}
