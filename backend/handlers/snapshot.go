package handlers

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Konstanta tipe event snapshot serta pengaturan direktori & rotasi file.
const (
	// EventStatusChanged dipakai saat status CCTV berubah.
	EventStatusChanged = "status_changed"
	// EventMaintenanceCompleted dipakai saat pemeliharaan diselesaikan.
	EventMaintenanceCompleted = "maintenance_completed"
	// EventFieldCheckAttention dipakai saat hasil pengecekan lapangan memerlukan perhatian.
	EventFieldCheckAttention = "field_check_attention"
	// EventPeriodicSnapshot dipakai oleh penjadwal otomatis berkala.
	EventPeriodicSnapshot = "periodic_snapshot"

	// snapshotDir adalah direktori relatif tempat file snapshot disimpan.
	snapshotDir = "uploads/snapshots"
	// maxSnapshotsPerCCTV membatasi jumlah file snapshot yang disimpan per CCTV.
	maxSnapshotsPerCCTV = 10
)

// snapshotClient adalah klien global MediaMTX untuk pengambilan snapshot.
var snapshotClient = utils.NewMediaMTXClient()

// snapshotEndpointUnavailable menandai bahwa MediaMTX yang berjalan tidak
// menyediakan endpoint snapshot (v1.20 tidak memilikinya). Scheduler dihentikan
// agar tidak menulis puluhan error 404 setiap siklus.
var snapshotEndpointUnavailable atomic.Bool

// snapshotURLFor membangun path URL snapshot terbaru sebuah CCTV.
func snapshotURLFor(cctvID string) string {
	return "/api/cctv/" + cctvID + "/snapshot"
}

// CaptureSnapshot mengambil snapshot JPEG dari MediaMTX untuk sebuah CCTV,
// menyimpannya ke disk, melakukan rotasi (membatasi jumlah file), lalu
// mencatat baris CCTVSnapshot ke database. Mengembalikan path relatif gambar.
// Pada kegagalan apa pun hanya mencatat peringatan dan TIDAK menyebabkan
// crash — dipakai dari goroutine scheduler maupun hook status.
func CaptureSnapshot(cctv models.CCTV, eventType, notes string) (string, error) {
	path := utils.BuildStreamPath(cctv.ID.String())
	data, err := snapshotClient.GetSnapshot(path)
	if err != nil {
		if isMediaMTXNotFound(err) {
			// Path bisa saja belum terdaftar atau endpoint snapshot memang tidak
			// tersedia. Scheduler menangani kondisi ini tanpa spam log.
			utils.Log.Debugf("Snapshot CCTV %s (%s) belum tersedia: %v", cctv.Nama, cctv.KodeCCTV, err)
		} else {
			utils.Log.Warnf("⚠️ Gagal mengambil snapshot CCTV %s (%s): %v", cctv.Nama, cctv.KodeCCTV, err)
		}
		return "", err
	}

	// Direktori per CCTV: uploads/snapshots/{cctv_id}
	cctvDir := filepath.Join(snapshotDir, cctv.ID.String())
	if err := os.MkdirAll(cctvDir, os.ModePerm); err != nil {
		utils.Log.Warnf("⚠️ Gagal membuat direktori snapshot %s: %v", cctvDir, err)
		return "", err
	}

	// Nama file berbasis timestamp: {timestamp}.jpg
	fileName := time.Now().Format("20060102_150405") + ".jpg"
	relPath := filepath.Join(cctvDir, fileName)
	if err := os.WriteFile(relPath, data, 0644); err != nil {
		utils.Log.Warnf("⚠️ Gagal menulis file snapshot %s: %v", relPath, err)
		return "", err
	}

	// Rotasi: hapus file tertua bila jumlah melebihi batas per CCTV.
	rotateSnapshots(cctvDir)

	// Catat metadata snapshot ke database.
	row := models.CCTVSnapshot{
		CCTVID:    cctv.ID,
		EventType: eventType,
		ImagePath: relPath,
		Notes:     notes,
	}
	if err := config.DB.Create(&row).Error; err != nil {
		utils.Log.Warnf("⚠️ Gagal menyimpan record snapshot CCTV %s: %v", cctv.ID, err)
	}

	return relPath, nil
}

// rotateSnapshots menghapus file JPEG tertua dalam direktori sehingga jumlah
// file per CCTV tidak melebihi maxSnapshotsPerCCTV. Berjalan best-effort.
func rotateSnapshots(cctvDir string) {
	entries, err := os.ReadDir(cctvDir)
	if err != nil {
		utils.Log.Warnf("⚠️ Gagal membaca direktori snapshot %s: %v", cctvDir, err)
		return
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".jpg" {
			files = append(files, filepath.Join(cctvDir, e.Name()))
		}
	}
	if len(files) <= maxSnapshotsPerCCTV {
		return
	}

	// Urutkan berdasarkan waktu modifikasi (tertua dulu) lalu hapus kelebihan.
	// Sederhana: gunakan nama file (timestamp) untuk urutan deterministik.
	excess := len(files) - maxSnapshotsPerCCTV
	removed := 0
	for _, f := range files {
		if removed >= excess {
			break
		}
		if err := os.Remove(f); err != nil {
			utils.Log.Warnf("⚠️ Gagal menghapus file snapshot lama %s: %v", f, err)
			continue
		}
		removed++
	}
}

// GetCCTVSnapshot menampilkan snapshot JPEG terbaru sebuah CCTV.
// @Summary      Ambil Snapshot CCTV
// @Description  Mengembalikan gambar JPEG snapshot terbaru dari sebuah CCTV yang diambil dari MediaMTX.
// @Tags         Snapshot
// @Produce      image/jpeg
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {string} string "Gambar JPEG snapshot"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      404 {object} map[string]interface{} "CCTV atau snapshot tidak ditemukan"
// @Router       /cctv/{id}/snapshot [get]
func GetCCTVSnapshot(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Data CCTV tidak ditemukan",
		})
	}

	// Ambil snapshot terbaru (urutkan created_at desc).
	var snap models.CCTVSnapshot
	if err := config.DB.Where("cctv_id = ?", id).Order("created_at desc").First(&snap).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Snapshot CCTV tidak ditemukan",
		})
	}

	// Pastikan file benar-benar ada di disk; bila hilang anggap 404.
	if _, err := os.Stat(snap.ImagePath); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "File snapshot tidak ditemukan di penyimpanan",
		})
	}

	c.Set("Content-Type", "image/jpeg")
	c.Set("Content-Disposition", "inline; filename=snapshot.jpg")
	return c.SendFile(snap.ImagePath)
}

// GetCCTVSnapshotStatus melaporkan apakah snapshot terbaru sebuah CCTV dapat
// ditampilkan. Frontend memakai endpoint ini untuk menonaktifkan tombol
// snapshot ketika MediaMTX tidak menyediakan endpoint snapshot (mis. v1.20)
// atau belum ada file snapshot, sehingga UI tidak menampilkan tombol yang
// selalu gagal.
// @Summary      Status Snapshot CCTV
// @Description  Mengembalikan ketersediaan snapshot terbaru, dukungan endpoint MediaMTX, waktu pengambilan, dan pesan alasan bila tidak tersedia.
// @Tags         Snapshot
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {object} map[string]interface{} "Status snapshot"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Router       /cctv/{id}/snapshot-status [get]
func GetCCTVSnapshotStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.Select("id").First(&cctv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Data CCTV tidak ditemukan",
		})
	}

	endpointSupported := !snapshotEndpointUnavailable.Load()
	data := fiber.Map{
		"available":          false,
		"endpoint_supported": endpointSupported,
		"snapshot_url":       snapshotURLFor(id.String()),
		"message":            "",
	}

	if !endpointSupported {
		data["message"] = "MediaMTX yang berjalan tidak menyediakan endpoint snapshot. Snapshot berkala dinonaktifkan."
		return c.JSON(fiber.Map{"status": "success", "data": data})
	}

	var snap models.CCTVSnapshot
	if err := config.DB.Where("cctv_id = ?", id).Order("created_at desc").First(&snap).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			data["message"] = "Belum ada snapshot tersimpan untuk CCTV ini."
			return c.JSON(fiber.Map{"status": "success", "data": data})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memeriksa snapshot CCTV",
		})
	}

	if _, err := os.Stat(snap.ImagePath); err != nil {
		data["message"] = "File snapshot tidak ditemukan di penyimpanan."
		return c.JSON(fiber.Map{"status": "success", "data": data})
	}

	data["available"] = true
	data["captured_at"] = snap.CreatedAt
	data["event_type"] = snap.EventType
	data["message"] = "Snapshot tersedia."
	return c.JSON(fiber.Map{"status": "success", "data": data})
}

// GetCCTVTimeline menampilkan riwayat event visual (timeline) sebuah CCTV
// secara terpaginasi, diurutkan dari terbaru ke terlama.
// @Summary      Ambil Timeline CCTV
// @Description  Menampilkan daftar snapshot / event visual CCTV terpaginasi (urutan terbaru dulu) dengan metadata paginasi.
// @Tags         Snapshot
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 20)"
// @Success      200 {object} map[string]interface{} "Daftar timeline snapshot terpaginasi"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Router       /cctv/{id}/timeline [get]
func GetCCTVTimeline(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.Select("id").First(&cctv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Data CCTV tidak ditemukan",
		})
	}

	// Paginasi: default limit 20 untuk timeline.
	limit, page, sort := utils.GetPaginationParams(c, []string{"created_at"})

	var rows []models.CCTVSnapshot
	var totalRows int64

	if err := config.DB.Model(&models.CCTVSnapshot{}).Where("cctv_id = ?", id).Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung data timeline",
		})
	}

	offset := (page - 1) * limit
	if err := config.DB.Where("cctv_id = ?", id).Offset(offset).Limit(limit).Order(sort).Find(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data timeline",
		})
	}

	// Sertakan path URL snapshot untuk setiap baris agar frontend dapat
	// menampilkan gambar melalui endpoint snapshot yang dilindungi.
	result := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		result = append(result, fiber.Map{
			"id":           r.ID,
			"cctv_id":      r.CCTVID,
			"event_type":   r.EventType,
			"image_path":   r.ImagePath,
			"notes":        r.Notes,
			"snapshot_url": snapshotURLFor(r.CCTVID.String()),
			"created_at":   r.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       result,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// RunSnapshotScheduler menjalankan pengambilan snapshot otomatis untuk seluruh
// CCTV berstatus stream_enabled=true. Percobaan pertama ditunda agar registrasi
// path selesai, lalu berulang setiap 15 menit. Bila MediaMTX terbukti tidak
// menyediakan endpoint snapshot, scheduler berhenti dengan satu peringatan.
func RunSnapshotScheduler(db *gorm.DB) {
	if db == nil {
		utils.Log.Warn("⚠️ RunSnapshotScheduler: db nil — scheduler dihentikan")
		return
	}

	// Interval pengambilan snapshot berkala (15 menit).
	const interval = 15 * time.Minute

	// Beri waktu registrasi path MediaMTX selesai sebelum percobaan pertama.
	time.Sleep(10 * time.Second)

	for {
		if snapshotEndpointUnavailable.Load() {
			return
		}

		var cctvs []models.CCTV
		if err := db.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
			utils.Log.Errorf("❌ RunSnapshotScheduler: gagal memuat CCTV aktif: %v", err)
		} else {
			success := 0
			for _, cctv := range cctvs {
				if _, err := CaptureSnapshot(cctv, EventPeriodicSnapshot, ""); err != nil {
					// Bila path terdaftar tetapi endpoint snapshot tetap 404,
					// berarti MediaMTX tidak menyediakan API snapshot. Hentikan
					// scheduler sekali saja daripada mengulang error setiap 15 menit.
					if isMediaMTXNotFound(err) {
						path := utils.BuildStreamPath(cctv.ID.String())
						if _, pathErr := snapshotClient.GetPath(path); pathErr == nil {
							snapshotEndpointUnavailable.Store(true)
							utils.Log.Warn("⚠️ Snapshot berkala dihentikan: MediaMTX tidak menyediakan endpoint snapshot (404)")
							return
						}
					}
					continue
				}
				success++
			}
			utils.Log.Infof("📸 Snapshot berkala selesai: %d/%d CCTV berhasil", success, len(cctvs))
		}

		time.Sleep(interval)
	}
}
