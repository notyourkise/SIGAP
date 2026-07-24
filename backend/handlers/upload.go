package handlers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Ukuran file maksimal: 2 MB
const maxUploadSize = 2 * 1024 * 1024

// Tipe MIME yang diizinkan
var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// UploadFile mengunggah file gambar bukti pemeliharaan CCTV
// @Summary      Unggah Foto Pemeliharaan
// @Description  Mengunggah file gambar (JPEG/PNG/WebP, maks 2MB) sebagai bukti pemeliharaan CCTV. Mengembalikan URL path file yang bisa disimpan ke field image_url pada log pemeliharaan.
// @Tags         Upload
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "File gambar (JPEG/PNG/WebP, maks 2MB)"
// @Success      200 {object} map[string]interface{} "Upload sukses, mengembalikan URL file"
// @Failure      400 {object} map[string]interface{} "Tidak ada file / tipe tidak valid / ukuran melebihi batas"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan file"
// @Router       /upload [post]
func UploadFile(c *fiber.Ctx) error {
	// Ambil file dari form-data dengan key "file"
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "File tidak ditemukan. Pastikan field form-data bernama 'file'.",
		})
	}

	// Validasi ukuran file (maks 2MB)
	if file.Size > maxUploadSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Ukuran file terlalu besar. Maksimal %dMB.", maxUploadSize/1024/1024),
		})
	}

	// Validasi tipe konten berkas
	contentType := file.Header.Get("Content-Type")
	ext, allowed := allowedMIMETypes[contentType]

	// Fallback: cek dari ekstensi nama file jika Content-Type tidak akurat
	if !allowed {
		fileExt := strings.ToLower(filepath.Ext(file.Filename))
		switch fileExt {
		case ".jpg", ".jpeg":
			ext = ".jpg"
			allowed = true
		case ".png":
			ext = ".png"
			allowed = true
		case ".webp":
			ext = ".webp"
			allowed = true
		}
	}

	if !allowed {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Tipe file tidak diizinkan. Hanya menerima JPEG, PNG, dan WebP.",
		})
	}

	// Hasilkan nama file unik berbasis UUID untuk menghindari tabrakan nama
	uniqueFilename := uuid.New().String() + ext
	savePath := "./uploads/" + uniqueFilename

	// Simpan file ke direktori ./uploads
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan file ke server",
		})
	}

	// Kembalikan URL path yang bisa langsung digunakan sebagai image_url
	fileURL := "/uploads/" + uniqueFilename

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"url":      fileURL,
			"filename": uniqueFilename,
			"size":     file.Size,
			"type":     contentType,
		},
	})
}
