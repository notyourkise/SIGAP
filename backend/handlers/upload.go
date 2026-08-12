package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/image/webp"
	"gorm.io/gorm"

	"sigap-backend/models"
	"sigap-backend/utils"
)

// Ukuran file maksimal: 2 MB
const maxUploadSize = 2 * 1024 * 1024

// Tipe MIME yang diizinkan (input diizinkan; semua output dinormalisasi ke JPEG)
var allowedMIMETypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

const (
	// uploadsDir adalah lokasi fisik file lampiran
	uploadsDir = "./uploads"
	// thumbDir adalah lokasi fisik file thumbnail
	thumbDir = "./uploads/thumbnails"
	// uploadPrefix adalah prefix URL publik file lampiran
	uploadPrefix = "/api/uploads/"
	// thumbPrefix adalah prefix URL publik file thumbnail
	thumbPrefix = "/api/uploads/thumbnails/"
	// uploadGracePeriod adalah tenggang waktu sebelum file yatim dihapus
	// (mencegah penghapusan file yang baru diupload tapi belum disimpan ke log)
	uploadGracePeriod = 24 * time.Hour

	// maxDimension adalah batas dimensi terpanjang (lebar/tinggi) gambar fullsize
	// setelah dikompresi. Gambar lebih besar dari ini akan di-resize mempertahankan
	// aspect ratio agar ukuran file tetap terkendali.
	maxDimension = 1920
	// thumbWidth dan thumbHeight adalah ukuran maksimal thumbnail yang dihasilkan
	thumbWidth  = 400
	thumbHeight = 300
	// compressThreshold adalah ambang ukuran file asli (dalam byte) untuk memutuskan
	// apakah gambar kecil perlu di-resize atau cukup disimpan apa adanya.
	compressThreshold = 300 * 1024
)

// UploadFile mengunggah file gambar bukti pemeliharaan CCTV
// @Summary      Unggah Foto Pemeliharaan
// @Description  Mengunggah file gambar (JPEG/PNG/WebP, maks 2MB) sebagai bukti pemeliharaan CCTV. Gambar di-decode ulang dan dinormalisasi ke JPEG sehingga seluruh metadata EXIF (termasuk lokasi GPS) ikut terhapus demi privasi. Mengembalikan URL path file yang bisa disimpan ke field image_url pada log pemeliharaan.
// @Tags         Upload
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "File gambar (JPEG/PNG/WebP, maks 2MB)"
// @Success      200 {object} map[string]interface{} "Upload sukses, mengembalikan URL file"
// @Failure      400 {object} map[string]interface{} "Tidak ada file / tipe tidak valid / ukuran melebihi batas / gagal decode"
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

	// Buka file untuk validasi konten via magic bytes
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca file yang diunggah",
		})
	}
	defer func() { _ = src.Close() }()

	// Baca 512 byte pertama untuk deteksi MIME yang akurat via magic bytes
	sniff := make([]byte, 512)
	n, err := src.Read(sniff)
	if err != nil || n == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca konten file",
		})
	}
	detectedType := http.DetectContentType(sniff[:n])

	// SECURITY: Hanya percaya magic bytes, TIDAK ada fallback ke ekstensi nama file.
	// Ekstensi bisa dipalsukan oleh attacker (mis. script.php diubah jadi image.jpg).
	if !allowedMIMETypes[detectedType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Tipe file tidak diizinkan. Hanya menerima JPEG, PNG, dan WebP. Pastikan file yang diunggah adalah gambar yang valid.",
		})
	}

	// Baca seluruh konten untuk proses decode + strip metadata
	fullData := make([]byte, file.Size)
	if _, err := src.ReadAt(fullData, 0); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca isi file",
		})
	}

	// PRIVACY & SECURITY: Decode ulang gambar lalu re-encode ke JPEG.
	// Proses ini menghapus seluruh metadata EXIF (termasuk lokasi GPS yang
	// sering tertanam di foto kamera HP) sekaligus memastikan file yang
	// tersimpan di server adalah gambar yang valid, bukan skrip tersembunyi.
	img, err := decodeImage(fullData, detectedType)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "File gambar tidak valid atau rusak (gagal diproses).",
		})
	}

	// Normalisasi ke RGB (hapus saluran alpha) agar encode JPEG tidak berlatar hitam.
	imgForSave := flattenToRGB(img)

	// Hasilkan nama file unik berbasis UUID (selalu JPEG hasil normalisasi)
	uniqueFilename := uuid.New().String() + ".jpg"
	savePath := filepath.Join(uploadsDir, uniqueFilename)
	thumbPath := filepath.Join(thumbDir, uniqueFilename)

	// Simpan versi fullsize (di-resize bila melebihi maxDimension) dan hitung ukuran
	// hasil kompresi untuk dilaporkan ke header respons.
	compSize, err := saveCompressed(imgForSave, savePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan file ke server",
		})
	}

	// Selalu buat thumbnail untuk versi kecil (mis. daftar / preview).
	if err := generateThumbnail(imgForSave, thumbPath); err != nil {
		// Jika thumbnail gagal dibuat, tetap lanjut: file fullsize sudah valid.
		utils.Log.Warnf("⚠️ Gagal membuat thumbnail %s: %v", uniqueFilename, err)
	}

	// Laporkan ukuran asli vs hasil kompresi via header (dipakai log/UI).
	c.Set("X-Original-Size", strconv.FormatInt(file.Size, 10))
	c.Set("X-Compressed-Size", strconv.FormatInt(compSize, 10))

	// Kembalikan URL path yang dilindungi JWT (bukan akses static langsung)
	fileURL := uploadPrefix + uniqueFilename
	thumbURL := thumbPrefix + uniqueFilename

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"url":       fileURL,
			"filename":  uniqueFilename,
			"size":      compSize,
			"type":      "image/jpeg",
			"thumb_url": thumbURL,
		},
	})
}

// ServeUploadedFile menyajikan file gambar yang sudah diunggah, hanya untuk user terautentikasi
// @Summary      Ambil File Foto Pemeliharaan
// @Description  Mengambil file gambar yang sudah diunggah berdasarkan nama file. Memerlukan autentikasi JWT.
// @Tags         Upload
// @Produce      image/jpeg,image/png,image/webp
// @Security     BearerAuth
// @Param        filename path string true "Nama file gambar (UUID.ext)"
// @Success      200 {file} binary "File gambar"
// @Failure      400 {object} map[string]interface{} "Nama file tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      404 {object} map[string]interface{} "File tidak ditemukan"
// @Router       /uploads/{filename} [get]
func ServeUploadedFile(c *fiber.Ctx) error {
	filename := c.Params("filename")

	// Validasi nama file: hanya boleh berisi karakter aman (UUID + ekstensi gambar)
	// Mencegah path traversal attack (mis. ../../etc/passwd)
	if filename == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Nama file tidak valid",
		})
	}

	// Pastikan ekstensi hanya gambar yang diizinkan
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowedExts[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Tipe file tidak diizinkan",
		})
	}

	// Bersihkan nama file dan bentuk path absolut yang aman
	cleanFilename := filepath.Base(filename)
	filePath := filepath.Join(uploadsDir, cleanFilename)

	// Periksa apakah file ada
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "File tidak ditemukan",
		})
	}

	// Kirim file sebagai response stream
	return c.SendFile(filePath)
}

// ServeUploadedThumbnail menyajikan file thumbnail gambar yang sudah diunggah,
// hanya untuk user terautentikasi.
// @Summary      Ambil Thumbnail Foto Pemeliharaan
// @Description  Mengambil file thumbnail gambar yang sudah diunggah berdasarkan nama file. Memerlukan autentikasi JWT.
// @Tags         Upload
// @Produce      image/jpeg
// @Security     BearerAuth
// @Param        filename path string true "Nama file thumbnail (UUID.jpg)"
// @Success      200 {file} binary "File thumbnail"
// @Failure      400 {object} map[string]interface{} "Nama file tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      404 {object} map[string]interface{} "File tidak ditemukan"
// @Router       /uploads/thumbnails/{filename} [get]
func ServeUploadedThumbnail(c *fiber.Ctx) error {
	filename := c.Params("filename")

	// Validasi nama file: hanya boleh berisi karakter aman (UUID + ekstensi jpg).
	// Mencegah path traversal attack (mis. ../../etc/passwd).
	if filename == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Nama file tidak valid",
		})
	}

	// Pastikan ekstensi hanya jpg (thumbnail selalu disimpan sebagai JPEG).
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Tipe file tidak diizinkan",
		})
	}

	// Bersihkan nama file dan bentuk path absolut yang aman di dalam thumbDir.
	cleanFilename := filepath.Base(filename)
	filePath := filepath.Join(thumbDir, cleanFilename)

	// Periksa apakah file ada
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "File tidak ditemukan",
		})
	}

	// Kirim file sebagai response stream
	return c.SendFile(filePath)
}

// decodeImage mendecode gambar dari byte sesuai tipe MIME yang terdeteksi.
// Mengembalikan error bila data bukan gambar valid.
func decodeImage(data []byte, detectedType string) (image.Image, error) {
	if detectedType == "image/webp" {
		return webp.Decode(bytes.NewReader(data))
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// flattenToRGB mengompositkan gambar ke latar putih bila memiliki saluran alpha
// (mis. PNG transparan) sehingga hasil encode JPEG tidak berlatar hitam.
func flattenToRGB(img image.Image) image.Image {
	opaque, ok := img.(interface{ Opaque() bool })
	if ok && opaque.Opaque() {
		return img
	}
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, bounds, img, image.Point{}, draw.Over)
	return dst
}

// jpegEncode menulis gambar ke writer dalam format JPEG kualitas 85.
// Kualitas ini menjadi patokan seluruh proses kompresi fullsize.
func jpegEncode(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, &jpeg.Options{Quality: 85})
}

// saveCompressed menyimpan gambar versi fullsize ke dstPath. Bila dimensi gambar
// lebih besar dari maxDimension, gambar di-resize mempertahankan aspect ratio
// (menggunakan filtering Lanczos untuk hasil tajam) sebelum di-encode ke JPEG
// kualitas 85. Mengembalikan ukuran file yang tersimpan dalam byte.
func saveCompressed(img image.Image, dstPath string) (int64, error) {
	// Pastikan direktori tujuan tersedia (aman bila main.go belum membuatnya).
	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
		return 0, err
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Hanya resize bila dimensi melebihi batas maksimal, agar gambar kecil
	// tidak diturunkan kualitasnya secara tidak perlu.
	out := img
	if w > maxDimension || h > maxDimension {
		out = imaging.Resize(img, maxDimension, maxDimension, imaging.Lanczos)
	}

	var buf bytes.Buffer
	if err := jpegEncode(&buf, out); err != nil {
		return 0, err
	}
	if err := os.WriteFile(dstPath, buf.Bytes(), 0o644); err != nil {
		return 0, err
	}
	return int64(buf.Len()), nil
}

// generateThumbnail membuat gambar thumbnail berukuran maksimal thumbWidth x
// thumbHeight (fit, mempertahankan aspect ratio) lalu menyimpannya ke thumbPath
// sebagai JPEG kualitas 80.
func generateThumbnail(img image.Image, thumbPath string) error {
	// Pastikan direktori thumbnail tersedia (aman bila main.go belum membuatnya).
	if err := os.MkdirAll(thumbDir, os.ModePerm); err != nil {
		return err
	}

	thumb := imaging.Fit(img, thumbWidth, thumbHeight, imaging.Lanczos)
	f, err := os.Create(thumbPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return jpeg.Encode(f, thumb, &jpeg.Options{Quality: 80})
}

// StartUploadCleanup menjalankan goroutine pembersihan file upload yatim
// (file yang tidak direferensikan oleh log pemeliharaan) secara berkala.
func StartUploadCleanup(db *gorm.DB, interval time.Duration) {
	go func() {
		CleanupOrphanUploads(db)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			CleanupOrphanUploads(db)
		}
	}()
}

// CleanupOrphanUploads menghapus file di direktori uploads yang tidak
// direferensikan oleh image_url pada tabel maintenance_logs / field_maintenance_images
// dan sudah berumur melebihi masa tenggang (uploadGracePeriod).
func CleanupOrphanUploads(db *gorm.DB) {
	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		utils.Log.Warnf("⚠️ Gagal membaca direktori uploads saat cleanup: %v", err)
		return
	}

	var imageURLs []string
	if err := db.Model(&models.MaintenanceLog{}).Unscoped().
		Where("image_url IS NOT NULL AND image_url <> ''").
		Pluck("image_url", &imageURLs).Error; err != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil daftar image_url saat cleanup: %v", err)
		return
	}

	// Foto bukti pengecekan lapangan juga merupakan referensi yang sah
	var fieldImageURLs []string
	if err := db.Model(&models.FieldMaintenanceImage{}).Unscoped().
		Pluck("image_url", &fieldImageURLs).Error; err != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil daftar image_url pengecekan lapangan saat cleanup: %v", err)
		return
	}
	imageURLs = append(imageURLs, fieldImageURLs...)

	// Foto profil pengguna (users.profile_photo_url) juga merupakan referensi sah
	var avatarURLs []string
	if err := db.Model(&models.User{}).Unscoped().
		Where("profile_photo_url IS NOT NULL AND profile_photo_url <> ''").
		Pluck("profile_photo_url", &avatarURLs).Error; err != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil daftar profile_photo_url saat cleanup: %v", err)
		return
	}
	imageURLs = append(imageURLs, avatarURLs...)

	referenced := make(map[string]bool, len(imageURLs))
	for _, u := range imageURLs {
		if strings.HasPrefix(u, uploadPrefix) {
			referenced[strings.TrimPrefix(u, uploadPrefix)] = true
		}
	}

	threshold := time.Now().Add(-uploadGracePeriod)
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if referenced[e.Name()] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		// Jangan hapus file yang baru diupload (masih dalam masa tenggang)
		if info.ModTime().After(threshold) {
			continue
		}
		if err := os.Remove(filepath.Join(uploadsDir, e.Name())); err == nil {
			removed++
			utils.Log.Infof("🗑️ File upload yatim dihapus: %s", e.Name())
		}
	}
	if removed > 0 {
		utils.Log.Infof("🧹 Cleanup file upload selesai: %d file yatim dihapus", removed)
	}
}
