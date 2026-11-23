package handlers

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"os"
	"sigap-backend/aiassistant"
	"sigap-backend/config"
	"time"
)

func AskAIAssistant(c *fiber.Ctx) error {
	var req struct {
		Question string `json:"question"`
		Mode     string `json:"mode"`
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
	if isOperationalSummaryQuestion(req.Question) && config.DB != nil {
		siteStr, _ := c.Locals("site_id").(string)
		siteID, _ := uuid.Parse(siteStr)
		if summaryDoc, ok, sumErr := buildSevenDaySummaryDocument(ctx, config.DB, siteID, time.Now().UTC()); sumErr == nil && ok {
			docs = append(append([]aiassistant.Document(nil), docs...), summaryDoc)
		}
	}
	result, err := client.AnswerWithOptions(ctx, req.Question, docs, aiassistant.AnswerOptions{
		Mode:                   req.Mode,
		MaxSteps:               aiassistant.DefaultMaxAgentSteps,
		AllowMultiStepToolLoop: true,
	})
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"status": "error", "message": "Asisten AI belum tersedia. Periksa konfigurasi gateway dan model pada server."})
	}
	return c.JSON(fiber.Map{"status": "success", "data": result})
}
