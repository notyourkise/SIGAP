package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupAssetRoutes mendefinisikan rute untuk modul pengelolaan aset CCTV
func SetupAssetRoutes(router fiber.Router) {
	assetGroup := router.Group("/assets", middleware.Protected())

	// Rute yang dapat diakses oleh Admin & Teknisi
	assetGroup.Get("/", handlers.GetAllAssets)
	assetGroup.Get("/export", handlers.ExportAssets)
	assetGroup.Get("/:id", handlers.GetAssetByID)
	assetGroup.Get("/:id/qrcode", handlers.GetAssetQRCode)

	// Rute yang hanya dapat diakses oleh Admin
	assetGroup.Post("/", middleware.RequireRole("admin"), handlers.CreateAsset)
	assetGroup.Put("/:id", middleware.RequireRole("admin"), handlers.UpdateAsset)
	assetGroup.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteAsset)
}
