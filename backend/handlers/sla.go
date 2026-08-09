package handlers

import (
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateSLAPolicyRequest adalah body pembuatan kebijakan SLA (Admin saja).
type CreateSLAPolicyRequest struct {
	Nama             string `json:"nama" validate:"required,min=2,max=100"`
	JenisTugas       string `json:"jenis_tugas" validate:"required,oneof=corrective preventive field_check"`
	TargetJamRespon  int    `json:"target_jam_respon" validate:"required,gte=0"`
	TargetJamSelesai int    `json:"target_jam_selesai" validate:"required,gte=1"`
}

// UpdateSLAPolicyRequest adalah body pembaruan kebijakan SLA (Admin saja).
type UpdateSLAPolicyRequest struct {
	Nama             string `json:"nama" validate:"omitempty,min=2,max=100"`
	JenisTugas       string `json:"jenis_tugas" validate:"omitempty,oneof=corrective preventive field_check"`
	TargetJamRespon  *int   `json:"target_jam_respon" validate:"omitempty,gte=0"`
	TargetJamSelesai *int   `json:"target_jam_selesai" validate:"omitempty,gte=1"`
}

// GetSLAPolicies mengambil daftar kebijakan SLA (Admin & Teknisi yang login).
// @Summary      Ambil Semua Kebijakan SLA
// @Description  Mendapatkan daftar kebijakan SLA terpaginasi.
// @Tags         SLA
// @Produce      json
// @Security     BearerAuth
// @Param        search query string false "Pencarian berdasarkan nama atau jenis tugas"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Success      200 {object} map[string]interface{} "Daftar kebijakan SLA terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar kebijakan SLA"
// @Router       /sla-policies [get]
func GetSLAPolicies(c *fiber.Ctx) error {
	var items []models.SLAPolicy
	query := config.DB.Model(&models.SLAPolicy{})

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query = query.Where("nama ILIKE ? OR jenis_tugas ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	allowedSort := []string{"id", "nama", "jenis_tugas", "target_jam_respon", "target_jam_selesai", "created_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total kebijakan SLA",
		})
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar kebijakan SLA",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       items,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetSLAPolicyByID mengambil detail satu kebijakan SLA.
// @Summary      Ambil Kebijakan SLA Detail
// @Description  Mendapatkan detail satu kebijakan SLA berdasarkan UUID-nya.
// @Tags         SLA
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Kebijakan SLA UUID"
// @Success      200 {object} map[string]interface{} "Data kebijakan SLA ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Kebijakan SLA tidak ditemukan"
// @Router       /sla-policies/{id} [get]
func GetSLAPolicyByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID kebijakan SLA tidak valid",
		})
	}

	var item models.SLAPolicy
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Kebijakan SLA tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// CreateSLAPolicy membuat kebijakan SLA baru (Admin saja).
// @Summary      Tambah Kebijakan SLA
// @Description  Membuat master kebijakan SLA baru (nama, jenis tugas, target waktu).
// @Tags         SLA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateSLAPolicyRequest true "Data Kebijakan SLA Baru"
// @Success      201 {object} map[string]interface{} "Kebijakan SLA berhasil disimpan"
// @Failure      400 {object} map[string]interface{} "Format input salah atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan kebijakan SLA ke database"
// @Router       /sla-policies [post]
func CreateSLAPolicy(c *fiber.Ctx) error {
	var req CreateSLAPolicyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	item := models.SLAPolicy{
		Nama:             strings.TrimSpace(req.Nama),
		JenisTugas:       req.JenisTugas,
		TargetJamRespon:  req.TargetJamRespon,
		TargetJamSelesai: req.TargetJamSelesai,
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan kebijakan SLA ke database",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_SLA_POLICY", "SLA",
		"Membuat kebijakan SLA baru: "+item.Nama, nil, "", "")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// UpdateSLAPolicy memperbarui kebijakan SLA (Admin saja).
// @Summary      Update Kebijakan SLA
// @Description  Memperbarui nama, jenis tugas, dan/atau target waktu kebijakan SLA.
// @Tags         SLA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Kebijakan SLA UUID"
// @Param        request body UpdateSLAPolicyRequest true "Data Update Kebijakan SLA"
// @Success      200 {object} map[string]interface{} "Kebijakan SLA berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Kebijakan SLA tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui kebijakan SLA"
// @Router       /sla-policies/{id} [put]
func UpdateSLAPolicy(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID kebijakan SLA tidak valid",
		})
	}

	var item models.SLAPolicy
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Kebijakan SLA tidak ditemukan",
		})
	}

	var req UpdateSLAPolicyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	if req.Nama != "" {
		item.Nama = strings.TrimSpace(req.Nama)
	}
	if req.JenisTugas != "" {
		item.JenisTugas = req.JenisTugas
	}
	if req.TargetJamRespon != nil {
		item.TargetJamRespon = *req.TargetJamRespon
	}
	if req.TargetJamSelesai != nil {
		item.TargetJamSelesai = *req.TargetJamSelesai
	}

	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui kebijakan SLA",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_SLA_POLICY", "SLA",
		"Memperbarui kebijakan SLA: "+item.Nama, nil, "", "")

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// DeleteSLAPolicy menghapus kebijakan SLA secara soft-delete (Admin saja).
// Jadwal yang masih menunjuk kebijakan ini otomatis dilepas (ON DELETE SET NULL).
// @Summary      Hapus Kebijakan SLA
// @Description  Menghapus kebijakan SLA (soft-delete). Jadwal yang menunjuk kebijakan ini otomatis dilepas.
// @Tags         SLA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Kebijakan SLA UUID"
// @Success      200 {object} map[string]interface{} "Kebijakan SLA berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Kebijakan SLA tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus kebijakan SLA"
// @Router       /sla-policies/{id} [delete]
func DeleteSLAPolicy(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID kebijakan SLA tidak valid",
		})
	}

	var item models.SLAPolicy
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Kebijakan SLA tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus kebijakan SLA",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_SLA_POLICY", "SLA",
		"Menghapus kebijakan SLA: "+item.Nama, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Kebijakan SLA berhasil dihapus",
	})
}

// ---- SLA Achievement Dashboard ----

// SLAMonthlyPoint adalah compliance SLA per bulan untuk chart bar.
type SLAMonthlyPoint struct {
	Month            string  `json:"month"`
	Total            int64   `json:"total"`
	CompletedInTarget int64  `json:"completed_in_target"`
	ComplianceRate   float64 `json:"compliance_rate"`
}

// SLAViolationItem adalah satu pelanggaran SLA (tugas melewati target waktu).
type SLAViolationItem struct {
	ID             string  `json:"id"`
	CCTVName       string  `json:"cctv_name"`
	CCTVCode       string  `json:"cctv_kode"`
	TechnicianName string  `json:"technician_name"`
	ActionTaken    string  `json:"action_taken"`
	Status         string  `json:"status"`
	TargetJamSelesai int   `json:"target_jam_selesai"`
	Deadline       *string `json:"deadline"`
	CompletedAt    *string `json:"completed_at"`
	OverdueHours   float64 `json:"overdue_hours"`
	JenisTugas     string  `json:"jenis_tugas"`
}

// SLATechnicianRank adalah ranking teknisi berdasarkan kepatuhan SLA.
type SLATechnicianRank struct {
	UserID          string  `json:"user_id"`
	FullName        string  `json:"full_name"`
	TotalTasks      int64   `json:"total_tasks"`
	CompletedInTarget int64 `json:"completed_in_target"`
	ComplianceRate  float64 `json:"compliance_rate"`
}

// slaLogRow adalah baris gabungan log + jadwal + kebijakan SLA untuk perhitungan.
type slaLogRow struct {
	ID               string
	CCTVID           string
	CCTVName         string
	CCTVCode         string
	TechnicianID     *string
	TechnicianName   string
	ActionTaken      string
	Status           string
	CreatedAt        time.Time
	CompletedAt      *time.Time
	TargetJamSelesai int
	JenisTugas       string
}

// GetDashboardSLA menghitung SLA achievement rate untuk dashboard admin.
// Compliance dihitung terhadap target penyelesaian (TargetJamSelesai) sejak log
// dibuat (created_at). Sebuah log dianggap "dalam target" bila diselesaikan
// sebelum deadline (created_at + target_jam_selesai jam).
// @Summary      Dashboard SLA
// @Description  Mengembalikan compliance SLA bulanan, daftar pelanggaran SLA, dan ranking teknisi untuk dashboard admin.
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Param        months query integer false "Jumlah bulan tren (default: 6)"
// @Success      200 {object} map[string]interface{} "Data dashboard SLA"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Hanya admin yang dapat mengakses dashboard SLA"
// @Failure      500 {object} map[string]interface{} "Gagal memuat data SLA"
// @Router       /dashboard/sla [get]
func GetDashboardSLA(c *fiber.Ctx) error {
	if role, _ := c.Locals("role").(string); role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: dashboard SLA hanya untuk admin",
		})
	}

	now := time.Now()
	months := 6
	if m := c.QueryInt("months"); m > 0 && m <= 24 {
		months = m
	}
	start := time.Date(now.Year(), now.Month()-time.Month(months-1), 1, 0, 0, 0, 0, now.Location())

	// Ambil seluruh log yang punya kebijakan SLA melalui maintenance_schedule.
	var rows []slaLogRow
	err := config.DB.Table("maintenance_logs AS ml").
		Select(`ml.id, ml.cctv_id, c.nama AS cctv_name, c.kode_cctv AS cctv_code,
			ml.technician_id, u.full_name AS technician_name, ml.action_taken, ml.status,
			ml.created_at, ml.completed_at, sp.target_jam_selesai, sp.jenis_tugas`).
		Joins("JOIN maintenance_schedules AS ms ON ms.id = ml.maintenance_schedule_id").
		Joins("JOIN sla_policies AS sp ON sp.id = ms.sla_policy_id").
		Joins("LEFT JOIN cctvs AS c ON c.id = ml.cctv_id").
		Joins("LEFT JOIN users AS u ON u.id = ml.technician_id").
		Where("ml.deleted_at IS NULL AND ml.created_at >= ?", start).
		Order("ml.created_at asc").
		Scan(&rows).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat data SLA",
		})
	}

	// Inisialisasi 6 slot bulan.
	monthKeys := make([]string, 0, months)
	monthly := map[string]*SLAMonthlyPoint{}
	for i := 0; i < months; i++ {
		m := time.Date(start.Year(), start.Month()+time.Month(i), 1, 0, 0, 0, 0, now.Location())
		key := m.Format("2006-01")
		monthKeys = append(monthKeys, key)
		monthly[key] = &SLAMonthlyPoint{Month: key}
	}

	violations := make([]SLAViolationItem, 0)
	techMap := map[string]*SLATechnicianRank{}

	for _, row := range rows {
		// Log dianggap bagian SLA hanya jika punya target penyelesaian.
		if row.TargetJamSelesai <= 0 {
			continue
		}
		deadline := row.CreatedAt.Add(time.Duration(row.TargetJamSelesai) * time.Hour)
		completed := row.CompletedAt != nil

		// Termasuk dalam compliance hanya log yang sudah terminal (completed/failed)
		// ATAU masih aktif namun sudah melewati deadline (dihitung pelanggaran).
		isViolation := false
		if completed {
			isViolation = row.CompletedAt.After(deadline)
		} else if row.Status == "ongoing" || row.Status == "scheduled" {
			isViolation = now.After(deadline)
		}

		// Catat ke bulan sesuai created_at.
		monthKey := row.CreatedAt.Format("2006-01")
		if pt, ok := monthly[monthKey]; ok {
			pt.Total++
			if completed && !isViolation {
				pt.CompletedInTarget++
			}
		}

		// Ranking teknisi.
		if row.TechnicianID != nil {
			tid := *row.TechnicianID
			tr, ok := techMap[tid]
			if !ok {
				tr = &SLATechnicianRank{UserID: tid, FullName: row.TechnicianName}
				techMap[tid] = tr
			}
			tr.TotalTasks++
			if completed && !isViolation {
				tr.CompletedInTarget++
			}
		}

		// Kumpulkan pelanggaran (batasi 100 terbaru).
		if isViolation {
			overdueHours := float64(0)
			if completed {
				overdueHours = row.CompletedAt.Sub(deadline).Hours()
			} else {
				overdueHours = now.Sub(deadline).Hours()
			}
			deadlineStr := deadline.Format("2006-01-02 15:04:05")
			completedStr := (*string)(nil)
			if row.CompletedAt != nil {
				s := row.CompletedAt.Format("2006-01-02 15:04:05")
				completedStr = &s
			}
			violations = append(violations, SLAViolationItem{
				ID:               row.ID,
				CCTVName:         row.CCTVName,
				CCTVCode:         row.CCTVCode,
				TechnicianName:   row.TechnicianName,
				ActionTaken:      row.ActionTaken,
				Status:           row.Status,
				TargetJamSelesai: row.TargetJamSelesai,
				Deadline:         &deadlineStr,
				CompletedAt:      completedStr,
				OverdueHours:     roundDashboardMetric(overdueHours),
				JenisTugas:       row.JenisTugas,
			})
		}
	}

	// Urutkan pelanggaran dari overdue terbesar.
	sortSLAViolations(violations)
	if len(violations) > 100 {
		violations = violations[:100]
	}

	// Bangun hasil bulanan & ranking.
	monthlyResult := make([]SLAMonthlyPoint, 0, months)
	for _, key := range monthKeys {
		pt := monthly[key]
		if pt.Total > 0 {
			pt.ComplianceRate = roundDashboardMetric(float64(pt.CompletedInTarget) / float64(pt.Total) * 100)
		}
		monthlyResult = append(monthlyResult, *pt)
	}

	ranking := make([]SLATechnicianRank, 0, len(techMap))
	for _, tr := range techMap {
		if tr.TotalTasks > 0 {
			tr.ComplianceRate = roundDashboardMetric(float64(tr.CompletedInTarget) / float64(tr.TotalTasks) * 100)
		}
		ranking = append(ranking, *tr)
	}
	// Urutkan ranking dari kepatuhan tertinggi.
	for i := 1; i < len(ranking); i++ {
		for j := i; j > 0 && ranking[j-1].ComplianceRate < ranking[j].ComplianceRate; j-- {
			ranking[j-1], ranking[j] = ranking[j], ranking[j-1]
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"monthly_compliance": monthlyResult,
			"violations":         violations,
			"technician_ranking": ranking,
		},
	})
}

// sortSLAViolations mengurutkan pelanggaran SLA dari yang paling lama overdue.
func sortSLAViolations(items []SLAViolationItem) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j-1].OverdueHours < items[j].OverdueHours; j-- {
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}

// ---- SLA Warning Goroutine ----

// EventSLAWarning adalah nama event WebSocket yang dipancarkan ketika sebuah
// tugas SLA telah melewati 75% dari target waktu penyelesaian (Task 11.5).
const EventSLAWarning = "SLA_WARNING"

// RunSLADeadlineChecker menjalankan pemeriksaan deadline SLA secara berkala.
// Berjalan sekali saat startup, lalu berulang tiap 1 jam. Memicu event
// SLA_WARNING ke seluruh admin bila ada tugas yang melewati 75% target.
func RunSLADeadlineChecker(db *gorm.DB) {
	checkSLADeadlinesOnce(db)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		checkSLADeadlinesOnce(db)
	}
}

// checkSLADeadlinesOnce melakukan satu siklus pemeriksaan deadline SLA.
func checkSLADeadlinesOnce(db *gorm.DB) {
	CheckSLADeadlines(db)
}

// CheckSLADeadlines memeriksa tugas yang sedang berjalan dan sudah melewati
// 75% target SLA untuk memicu peringatan. Dipanggil dari goroutine berkala
// (mis. setiap jam) di main.go. Idempotent terhadap log yang sudah selesai.
func CheckSLADeadlines(db *gorm.DB) {
	if db == nil {
		return
	}

	now := time.Now()

	// Cari log yang masih aktif (scheduled/ongoing), punya jadwal, dan jadwalnya
	// memiliki kebijakan SLA.
	var rows []struct {
		ID               string
		CCTVID           string
		CCTVName         string
		ActionTaken      string
		TechnicianID     *string
		CreatedAt        time.Time
		TargetJamSelesai int
		JenisTugas       string
	}
	err := db.Table("maintenance_logs AS ml").
		Select(`ml.id, ml.cctv_id, c.nama AS cctv_name, ml.action_taken, ml.technician_id,
			ml.created_at, sp.target_jam_selesai, sp.jenis_tugas`).
		Joins("JOIN maintenance_schedules AS ms ON ms.id = ml.maintenance_schedule_id").
		Joins("JOIN sla_policies AS sp ON sp.id = ms.sla_policy_id").
		Joins("LEFT JOIN cctvs AS c ON c.id = ml.cctv_id").
		Where("ml.deleted_at IS NULL AND ml.status IN ? AND sp.target_jam_selesai > 0",
			[]string{"scheduled", "ongoing"}).
		Scan(&rows).Error
	if err != nil {
		utils.Log.Errorf("❌ Gagal memuat tugas SLA untuk pemeriksaan deadline: %v", err)
		return
	}

	for _, row := range rows {
		deadline := row.CreatedAt.Add(time.Duration(row.TargetJamSelesai) * time.Hour)
		elapsed := now.Sub(row.CreatedAt)
		total := time.Duration(row.TargetJamSelesai) * time.Hour
		if total <= 0 {
			continue
		}
		// Hitung persentase waktu terpakai.
		pct := elapsed.Seconds() / total.Seconds() * 100

		// Peringatan saat tepat 75% terpakai dan belum melewati batas (agar tidak
		// terus-terusan). Kirim sekali per tugas: gunakan log waktu ke-1 dan ke-2
		// agar tidak spam; batasi window 75%–100%.
		if pct >= 75 && pct < 100 {
			utils.BroadcastToRole("admin", EventSLAWarning, fiber.Map{
				"log_id":        row.ID,
				"cctv_id":       row.CCTVID,
				"cctv_name":     row.CCTVName,
				"action_taken":  row.ActionTaken,
				"jenis_tugas":   row.JenisTugas,
				"target_jam":    row.TargetJamSelesai,
				"elapsed_hours": roundDashboardMetric(elapsed.Hours()),
				"deadline":      deadline.Format(time.RFC3339),
			})
		}
	}
}
