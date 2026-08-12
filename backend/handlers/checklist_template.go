package handlers

import (
	"fmt"
	"strings"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateChecklistTemplateRequest adalah body pembuatan item template checklist
// baru (Admin saja). nama_item bersifat wajib (minimal 1 karakter).
type CreateChecklistTemplateRequest struct {
	NamaItem  string `json:"nama_item" validate:"required,min=1,max=255"`
	Deskripsi string `json:"deskripsi" validate:"omitempty,max=2000"`
	Urutan    int    `json:"urutan" validate:"gte=0"`
	Wajib     bool   `json:"wajib"`
	Aktif     bool   `json:"aktif"`
}

// UpdateChecklistTemplateRequest adalah body pembaruan item template checklist
// (Admin saja). Semua field bersifat opsional.
type UpdateChecklistTemplateRequest struct {
	NamaItem  string `json:"nama_item" validate:"omitempty,min=1,max=255"`
	Deskripsi string `json:"deskripsi" validate:"omitempty,max=2000"`
	Urutan    *int   `json:"urutan" validate:"omitempty,gte=0"`
	Wajib     *bool  `json:"wajib"`
	Aktif     *bool  `json:"aktif"`
}

// ReorderChecklistTemplateItem merepresentasikan satu entri urutan dalam
// payload reorder (Admin saja).
type ReorderChecklistTemplateItem struct {
	ID     string `json:"id" validate:"required,uuid4"`
	Urutan int    `json:"urutan" validate:"gte=0"`
}

// ReorderChecklistTemplatesRequest adalah body untuk mengubah urutan banyak
// item template checklist sekaligus.
type ReorderChecklistTemplatesRequest struct {
	Items []ReorderChecklistTemplateItem `json:"items" validate:"required,min=1,dive"`
}

// GetAllChecklistTemplates mengambil daftar template checklist pengecekan
// terpaginasi, dapat difilter berdasarkan status aktif dan diurutkan default
// berdasarkan urutan menaik. Dapat diakses seluruh role yang sudah login.
// @Summary      Ambil Semua Template Checklist
// @Description  Mendapatkan daftar template checklist pengecekan lapangan yang dapat dikustomisasi admin. Mendukung filter aktif (?aktif=true/false) dan paginasi (?page=&limit=). Diurutkan berdasarkan urutan menaik.
// @Tags         Template Checklist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        aktif query string false "Filter berdasarkan status aktif (true/false)"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Success      200 {object} map[string]interface{} "Daftar template checklist terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar template checklist"
// @Router       /checklist-templates [get]
func GetAllChecklistTemplates(c *fiber.Ctx) error {
	var items []models.ChecklistTemplate
	query := config.DB.Model(&models.ChecklistTemplate{})

	// Filter status aktif (opsional): ?aktif=true → hanya item aktif.
	if aktif := c.Query("aktif"); aktif != "" {
		query = query.Where("aktif = ?", aktif == "true")
	}

	limit, page, _ := utils.GetPaginationParams(c, []string{"id", "urutan", "nama_item", "created_at"})

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total template checklist",
		})
	}

	offset := (page - 1) * limit
	// Default urutkan berdasarkan urutan menaik agar sesuai tampilan checklist.
	if err := query.Offset(offset).Limit(limit).Order("urutan ASC").Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar template checklist",
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

// CreateChecklistTemplate membuat item template checklist baru (Admin saja).
// Nama item wajib diisi. Default aktif dan wajib bernilai true.
// @Summary      Tambah Template Checklist
// @Description  Membuat item template checklist pengecekan lapangan baru yang dapat dikustomisasi admin. Field aktif dan wajib default ke true bila tidak dikirim.
// @Tags         Template Checklist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateChecklistTemplateRequest true "Data Template Checklist Baru"
// @Success      201 {object} map[string]interface{} "Template checklist berhasil dibuat"
// @Failure      400 {object} map[string]interface{} "Format input salah atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan template checklist"
// @Router       /checklist-templates [post]
func CreateChecklistTemplate(c *fiber.Ctx) error {
	var req CreateChecklistTemplateRequest
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

	// Default: aktif dan wajib true bila tidak dikirim client.
	aktif := req.Aktif
	if !req.Aktif && !bodyHasField(c, "aktif") {
		aktif = true
	}
	wajib := req.Wajib
	if !req.Wajib && !bodyHasField(c, "wajib") {
		wajib = true
	}

	item := models.ChecklistTemplate{
		NamaItem:  strings.TrimSpace(req.NamaItem),
		Deskripsi: strings.TrimSpace(req.Deskripsi),
		Urutan:    req.Urutan,
		Wajib:     wajib,
		Aktif:     aktif,
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan template checklist",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_CHECKLIST_TEMPLATE", "MAINTENANCE",
		"Membuat template checklist: "+item.NamaItem, nil, "", "")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// UpdateChecklistTemplate memperbarui item template checklist (Admin saja).
// Seluruh field (nama_item, deskripsi, urutan, wajib, aktif) dapat diubah.
// @Summary      Update Template Checklist
// @Description  Memperbarui item template checklist pengecekan lapangan. Hanya field yang dikirim yang akan diubah.
// @Tags         Template Checklist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Template Checklist UUID"
// @Param        request body UpdateChecklistTemplateRequest true "Data Update Template Checklist"
// @Success      200 {object} map[string]interface{} "Template checklist berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Template checklist tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui template checklist"
// @Router       /checklist-templates/{id} [put]
func UpdateChecklistTemplate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID template checklist tidak valid",
		})
	}

	var item models.ChecklistTemplate
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Template checklist tidak ditemukan",
		})
	}

	var req UpdateChecklistTemplateRequest
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

	if req.NamaItem != "" {
		item.NamaItem = strings.TrimSpace(req.NamaItem)
	}
	if bodyHasField(c, "deskripsi") {
		item.Deskripsi = strings.TrimSpace(req.Deskripsi)
	}
	if req.Urutan != nil {
		item.Urutan = *req.Urutan
	}
	if req.Wajib != nil {
		item.Wajib = *req.Wajib
	}
	if req.Aktif != nil {
		item.Aktif = *req.Aktif
	}

	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui template checklist",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_CHECKLIST_TEMPLATE", "MAINTENANCE",
		"Memperbarui template checklist: "+item.NamaItem, nil, "", "")

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// DeleteChecklistTemplate menonaktifkan item template checklist (Admin saja).
// PENTING: soft delete dengan set aktif = false, bukan hard delete, agar
// template yang sudah dipakai tetap dapat ditelusuri.
// @Summary      Hapus Template Checklist
// @Description  Menonaktifkan (soft delete) item template checklist dengan mengatur aktif menjadi false. Data tidak dihapus permanen.
// @Tags         Template Checklist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Template Checklist UUID"
// @Success      200 {object} map[string]interface{} "Template checklist berhasil dinonaktifkan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Template checklist tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menonaktifkan template checklist"
// @Router       /checklist-templates/{id} [delete]
func DeleteChecklistTemplate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID template checklist tidak valid",
		})
	}

	var item models.ChecklistTemplate
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Template checklist tidak ditemukan",
		})
	}

	// Soft delete: hanya menonaktifkan, tidak menghapus data permanen.
	if err := config.DB.Model(&item).Update("aktif", false).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menonaktifkan template checklist",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_CHECKLIST_TEMPLATE", "MAINTENANCE",
		"Menonaktifkan template checklist: "+item.NamaItem, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Template checklist berhasil dinonaktifkan",
	})
}

// ReorderChecklistTemplates mengubah urutan banyak item template checklist
// sekaligus dalam satu transaksi (Admin saja). Body berisi daftar {id, urutan}.
// @Summary      Urutkan Template Checklist
// @Description  Memperbarui urutan banyak item template checklist sekaligus (batch update) dalam satu transaksi atomik.
// @Tags         Template Checklist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body ReorderChecklistTemplatesRequest true "Daftar Urutan Template Checklist"
// @Success      200 {object} map[string]interface{} "Urutan template checklist berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "Format body salah atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui urutan template checklist"
// @Router       /checklist-templates/reorder [put]
func ReorderChecklistTemplates(c *fiber.Ctx) error {
	var req ReorderChecklistTemplatesRequest
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

	// Lakukan batch update urutan dalam satu transaksi agar atomik.
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			id, err := uuid.Parse(item.ID)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "ID template checklist tidak valid: "+item.ID)
			}
			if err := tx.Model(&models.ChecklistTemplate{}).
				Where("id = ?", id).
				Update("urutan", item.Urutan).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{
				"status":  "error",
				"message": fe.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui urutan template checklist",
		})
	}

	utils.LogAudit(config.DB, c, "REORDER_CHECKLIST_TEMPLATE", "MAINTENANCE",
		fmt.Sprintf("Memperbarui urutan %d item template checklist", len(req.Items)), nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Urutan template checklist berhasil diperbarui",
	})
}

// bodyHasField memeriksa apakah sebuah field kunci hadir di dalam body JSON.
// Dipakai untuk membedakan field yang sengaja dikirim false/nol vs yang tidak
// dikirim sama sekali (agar default true tetap berlaku saat pembuatan).
func bodyHasField(c *fiber.Ctx, key string) bool {
	var raw map[string]interface{}
	if err := c.BodyParser(&raw); err != nil {
		return false
	}
	_, ok := raw[key]
	return ok
}
