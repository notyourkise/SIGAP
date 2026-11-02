package handlers

import (
	"github.com/gofiber/fiber/v2"
	"net/http/httptest"
	"testing"
)

func TestIncidentListRequiresExplicitSite(t *testing.T) {
	app := fiber.New()
	app.Get("/incidents", GetIncidents)
	response, err := app.Test(httptest.NewRequest("GET", "/incidents", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 400 {
		t.Fatalf("expected explicit-site rejection, got %d", response.StatusCode)
	}
}
