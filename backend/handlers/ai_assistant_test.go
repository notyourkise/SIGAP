package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsOperationalSummaryQuestion(t *testing.T) {
	positive := []string{
		"Rangkum CCTV yang mati dalam 7 hari terakhir beserta penyebabnya",
		"Kamera apa saja yang sering offline minggu ini?",
		"Kenapa kamera THD-21 sering gangguan?",
		"Berikan statistik dan penyebab tiket aktif saat ini",
	}
	for _, q := range positive {
		if !isOperationalSummaryQuestion(q) {
			t.Fatalf("expected %q to match operational summary", q)
		}
	}

	negative := []string{
		"Apa entrypoint resmi web SIGAP untuk deployment Docker?",
		"Bagaimana cara restore backup database?",
		"Apakah status standby selalu dicatat sebagai event log?",
	}
	for _, q := range negative {
		if isOperationalSummaryQuestion(q) {
			t.Fatalf("did not expect static doc question %q to trigger live operational summary", q)
		}
	}
}

func TestBuildSevenDaySummaryDocumentHandlesNilDB(t *testing.T) {
	doc, ok, err := buildSevenDaySummaryDocument(context.Background(), nil, uuid.Nil, time.Now().UTC())
	if err != nil || ok || doc.ID != "" {
		t.Fatalf("expected clean no-op for nil DB, got doc=%+v ok=%v err=%v", doc, ok, err)
	}
}
