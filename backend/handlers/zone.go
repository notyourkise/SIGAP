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

// CreateZoneRequest adalah body pembuatan zona baru (Admin saja).
type CreateZoneRequest struct {
	Nama      string             `json:"nama" validate:"required,min=2,max=100"`
	Deskripsi string             `json:"deskripsi" validate:"omitempty"`
	SiteID    utils.OptionalUUID `json:"site_id" validate:"omitempty"`
}

// UpdateZoneRequest adalah body pembaruan zona (Admin saja).
type UpdateZoneRequest struct {
	Nama      string             `json:"nama" validate:"omitempty,min=2,max=100"`
	Deskripsi string             `json:"deskripsi" validate:"omitempty"`
	SiteID    utils.OptionalUUID `json:"site_id" validate:"omitempty"`
}

// resolveActiveSiteID membaca UUID site aktif dari c.Locals("site_id") (hasil
// middleware SiteContext), atau fallback ke header X-Site-ID, cookie
// sigap_site_id, dan query ?site_id= bila handler dipanggil langsung pada unit
// test tanpa middleware SiteContext. Mengembalikan (nil, true) bila tidak ada
// konteks site aktif, atau (nil, false) bila format UUID tidak valid.
func resolveActiveSiteID(c *fiber.Ctx) (*uuid.UUID, bool) {
	raw, _ := c.Locals("site_id").(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = strings.TrimSpace(c.Get("X-Site-ID"))
	}
	if raw == "" {
		raw = strings.TrimSpace(c.Cookies("sigap_site_id"))
	}
	if raw == "" {
		raw = strings.TrimSpace(c.Query("site_id"))
	}
	if raw == "" {
		return nil, true
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

// AssignCCTVRequest adalah body bulk-assign CCTV ke sebuah zona (Admin saja).
type AssignCCTVRequest struct {
	// CCTVIDs adalah daftar ID CCTV yang akan ditetapkan ke zona.
	CCTVIDs []uuid.UUID `json:"cctv_ids" validate:"required"`
}

// AssignUsersRequest adalah body assign teknisi ke sebuah zona (Admin saja).
type AssignUsersRequest struct {
	// UserIDs adalah daftar ID user (teknisi) yang akan ditetapkan ke zona.
	UserIDs []uuid.UUID `json:"user_ids" validate:"required"`
}

// ZoneListRow adalah item daftar zona dengan jumlah CCTV & teknisi.
type ZoneListRow struct {
	models.Zone
	CCTVCount int64 `gorm:"-" json:"cctv_count"`
	UserCount int64 `gorm:"-" json:"user_count"`
}

// GetAllZones mengambil daftar zona terpaginasi (Admin & Teknisi yang login)
// beserta jumlah CCTV dan teknisi per zona.
// @Summary      Ambil Semua Zona
// @Description  Mendapatkan daftar zona operasional terpaginasi dengan pencarian nama dan jumlah CCTV/teknisi per zona.
// @Tags         Zona
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        search query string false "Pencarian berdasarkan nama zona"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar zona terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar zona"
// @Router       /zones [get]
func GetAllZones(c *fiber.Ctx) error {
	var items []models.Zone
	query := config.DB.Model(&models.Zone{})

	// Filter site (multi-site, Task 12.2): aktif bila c.Locals("site_id"),
	// header X-Site-ID, cookie sigap_site_id, atau query ?site_id= diberikan.
	activeSiteID, validSite := resolveActiveSiteID(c)
	if !validSite {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format site_id tidak valid",
		})
	}
	if activeSiteID != nil {
		query = query.Where("site_id = ?", activeSiteID.String())
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query = query.Where("nama ILIKE ?", "%"+search+"%")
	}

	allowedSort := []string{"id", "nama", "created_at", "updated_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total zona",
		})
	}

	offset := (page - 1) * limit
	// Ambil zona dasar terpaginasi, lalu isi jumlah CCTV & teknisi per zona.
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar zona",
		})
	}

	// Bangun baris respons dengan hitungan relasi.
	rows := make([]ZoneListRow, 0, len(items))
	for _, z := range items {
		var cctvCount, userCount int64
		cctvCountQuery := config.DB.Model(&models.CCTV{}).Where("zone_id = ?", z.ID)
		if z.SiteID != nil {
			cctvCountQuery = cctvCountQuery.Where("site_id = ?", *z.SiteID)
		}
		cctvCountQuery.Count(&cctvCount)
		config.DB.Table("user_zones").Where("zone_id = ?", z.ID).Count(&userCount)
		rows = append(rows, ZoneListRow{Zone: z, CCTVCount: cctvCount, UserCount: userCount})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       rows,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetAllZonesForSelect mengambil seluruh zona tanpa paginasi untuk dropdown
// filter (mis. filter peta per zona dan select assign). Dapat diakses semua role.
// @Summary      Ambil Semua Zona (Select)
// @Description  Mendapatkan seluruh zona (id + nama + jumlah CCTV) tanpa paginasi untuk keperluan dropdown/filter.
// @Tags         Zona
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Daftar zona (tanpa paginasi)"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar zona"
// @Router       /zones/all [get]
func GetAllZonesForSelect(c *fiber.Ctx) error {
	var items []models.Zone
	query := config.DB.Model(&models.Zone{})
	activeSiteID, validSite := resolveActiveSiteID(c)
	if !validSite {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format site_id tidak valid",
		})
	}
	if activeSiteID != nil {
		query = query.Where("site_id = ?", activeSiteID.String())
	}
	if err := query.Order("nama asc").Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar zona",
		})
	}
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   items,
	})
}

// GetZoneByID mengambil detail satu zona beserta relasi CCTV & teknisi.
// @Summary      Ambil Zona Detail
// @Description  Mendapatkan detail satu zona beserta daftar CCTV dan teknisi yang ditugaskan.
// @Tags         Zona
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Zona UUID"
// @Success      200 {object} map[string]interface{} "Data zona ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Zona tidak ditemukan"
// @Router       /zones/{id} [get]
func GetZoneByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID zona tidak valid",
		})
	}

	activeSiteID, validSite := resolveActiveSiteID(c)
	if !validSite {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format site_id tidak valid",
		})
	}

	var item models.Zone
	if activeSiteID != nil {
		if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Zona tidak ditemukan",
			})
		}
		if !zoneMatchesActiveSite(c, item) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: zona berada di site lain",
			})
		}
	}
	if err := config.DB.Preload("CCTVs").Preload("Users").
		First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Zona tidak ditemukan",
		})
	}
	if !zoneMatchesActiveSite(c, item) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: zona berada di site lain",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// CreateZone membuat zona baru (Admin saja).
// @Summary      Tambah Zona
// @Description  Membuat master zona operasional baru.
// @Tags         Zona
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateZoneRequest true "Data Zona Baru"
// @Success      201 {object} map[string]interface{} "Zona berhasil disimpan"
// @Failure      400 {object} map[string]interface{} "Format input salah atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan zona ke database"
// @Router       /zones [post]
func CreateZone(c *fiber.Ctx) error {
	var req CreateZoneRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	activeSiteID, validSite := resolveActiveSiteID(c)
	if !validSite {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format site_id tidak valid",
		})
	}

	item := models.Zone{
		Nama:      strings.TrimSpace(req.Nama),
		Deskripsi: req.Deskripsi,
	}

	if activeSiteID != nil {
		if req.SiteID.Present && req.SiteID.Null {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Zona pada konteks site aktif tidak boleh memiliki site_id kosong",
			})
		}
		if req.SiteID.HasValue() && *req.SiteID.Value != *activeSiteID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Site zona tidak sesuai dengan site aktif yang sedang dipilih",
			})
		}
		siteCopy := *activeSiteID
		item.SiteID = &siteCopy
	} else if req.SiteID.HasValue() {
		siteCopy := *req.SiteID.Value
		item.SiteID = &siteCopy
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan zona ke database",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_ZONE", "ZONE",
		"Membuat zona baru: "+item.Nama, nil, "", "")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// UpdateZone memperbarui data zona (Admin saja).
// @Summary      Update Zona
// @Description  Memperbarui nama dan/atau deskripsi zona.
// @Tags         Zona
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Zona UUID"
// @Param        request body UpdateZoneRequest true "Data Update Zona"
// @Success      200 {object} map[string]interface{} "Zona berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Zona tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui zona"
// @Router       /zones/{id} [put]
func UpdateZone(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID zona tidak valid",
		})
	}

	activeSiteID, validSite := resolveActiveSiteID(c)
	if !validSite {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format site_id tidak valid",
		})
	}

	var item models.Zone
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Zona tidak ditemukan",
		})
	}
	if !zoneMatchesActiveSite(c, item) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: zona berada di site lain",
		})
	}

	var req UpdateZoneRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
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
	if req.Deskripsi != "" {
		item.Deskripsi = req.Deskripsi
	}

	if req.SiteID.Present {
		if req.SiteID.Null {
			if activeSiteID != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "Tidak dapat mengosongkan site zona saat konteks site aktif dipilih",
				})
			}
			var boundCCTVs int64
			if err := config.DB.Model(&models.CCTV{}).
				Where("zone_id = ? AND site_id IS NOT NULL AND deleted_at IS NULL", item.ID).
				Count(&boundCCTVs).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "Gagal memvalidasi relasi CCTV pada zona",
				})
			}
			if boundCCTVs > 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "Zona masih memiliki CCTV terikat site; lepaskan CCTV terlebih dahulu",
				})
			}
			item.SiteID = nil
		} else if req.SiteID.HasValue() {
			if activeSiteID != nil && *req.SiteID.Value != *activeSiteID {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "Site zona tidak sesuai dengan site aktif yang sedang dipilih",
				})
			}
			if item.SiteID == nil || *item.SiteID != *req.SiteID.Value {
				var mismatchedCCTVs int64
				if err := config.DB.Model(&models.CCTV{}).
					Where("zone_id = ? AND (site_id IS NULL OR site_id <> ?) AND deleted_at IS NULL", item.ID, *req.SiteID.Value).
					Count(&mismatchedCCTVs).Error; err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"status":  "error",
						"message": "Gagal memvalidasi relasi CCTV pada zona",
					})
				}
				if mismatchedCCTVs > 0 {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
						"status":  "error",
						"message": "Zona tidak dapat dipindahkan ke site lain karena masih memiliki CCTV dari site berbeda",
					})
				}
			}
			siteCopy := *req.SiteID.Value
			item.SiteID = &siteCopy
		}
	}

	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui zona",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_ZONE", "ZONE",
		"Memperbarui zona: "+item.Nama, nil, "", "")

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// DeleteZone menghapus zona secara soft-delete (Admin saja). CCTV yang masih
// menunjuk zona ini otomatis dilepas (ON DELETE SET NULL).
// @Summary      Hapus Zona
// @Description  Menghapus zona (soft-delete). CCTV yang berada pada zona ini otomatis dilepas dari zona (zone_id menjadi null).
// @Tags         Zona
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Zona UUID"
// @Success      200 {object} map[string]interface{} "Zona berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Zona tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus zona"
// @Router       /zones/{id} [delete]
func DeleteZone(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID zona tidak valid",
		})
	}

	var item models.Zone
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Zona tidak ditemukan",
		})
	}
	if !zoneMatchesActiveSite(c, item) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: zona berada di site lain",
		})
	}

	// Gunakan transaksi agar pelepasan CCTV dan penugasan teknisi berjalan
	// bersama soft delete zona. Dengan ON DELETE SET NULL pada FK, menghapus
	// zona akan mengosongkan zone_id CCTV secara otomatis.
	if err := config.DB.Delete(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus zona",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_ZONE", "ZONE",
		"Menghapus zona: "+item.Nama, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Zona berhasil dihapus",
	})
}

// AssignCCTVToZone melakukan bulk-assign daftar CCTV ke sebuah zona. Metode ini
// menimpa seluruh penugasan CCTV pada zona tersebut dengan daftar yang dikirim
// (di-leak CCTVIDs terlebih dahulu dari zona lain). Admin saja.
// @Summary      Assign CCTV ke Zona
// @Description  Menetapkan daftar CCTV ke sebuah zona. Seluruh CCTV pada daftar akan di-set zone_id ke zona ini (menimpa penugasan zona lama bila ada).
// @Tags         Zona
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Zona UUID"
// @Param        request body AssignCCTVRequest true "Daftar ID CCTV"
// @Success      200 {object} map[string]interface{} "CCTV berhasil ditugaskan ke zona"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, daftar kosong, atau ada CCTV tidak ditemukan"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Zona tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menetapkan CCTV ke zona"
// @Router       /zones/{id}/assign-cctvs [post]
func AssignCCTVToZone(c *fiber.Ctx) error {
	zoneID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID zona tidak valid",
		})
	}

	var zone models.Zone
	if err := config.DB.First(&zone, "id = ?", zoneID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Zona tidak ditemukan",
		})
	}
	if !zoneMatchesActiveSite(c, zone) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: zona berada di site lain",
		})
	}

	var req AssignCCTVRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 || len(req.CCTVIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Daftar CCTV wajib diisi dan tidak boleh kosong",
		})
	}

	// Tentukan kumpulan ID unik terlebih dahulu.
	seen := map[uuid.UUID]bool{}
	uniqueIDs := make([]uuid.UUID, 0, len(req.CCTVIDs))
	var ids []interface{}
	for _, id := range req.CCTVIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
			ids = append(ids, id)
		}
	}

	// Konsistensi site zona dan CCTV: bila zona memiliki SiteID (atau ada site
	// aktif pada request), seluruh CCTV yang ditugaskan wajib berasal dari site
	// yang sama dengan zona tersebut.
	var requiredSiteID *uuid.UUID
	if zone.SiteID != nil {
		requiredSiteID = zone.SiteID
	} else if activeSiteID, ok := resolveActiveSiteID(c); ok && activeSiteID != nil {
		requiredSiteID = activeSiteID
	}

	// Verifikasi semua CCTV ada dan berada pada site yang konsisten.
	var found int64
	cctvCheckQuery := config.DB.Model(&models.CCTV{}).
		Where("id IN ? AND deleted_at IS NULL", uniqueIDs)
	if requiredSiteID != nil {
		cctvCheckQuery = cctvCheckQuery.Where("site_id = ?", *requiredSiteID)
	}
	if err := cctvCheckQuery.Count(&found).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memvalidasi daftar CCTV",
		})
	}
	if found != int64(len(uniqueIDs)) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Satu atau lebih CCTV pada daftar tidak ditemukan atau berada di site yang berbeda dengan zona",
		})
	}

	// Set zone_id CCTV. Gunakan transaksi untuk memastikan konsistensi.
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		updateQuery := tx.Model(&models.CCTV{}).Where("id IN ?", ids)
		if requiredSiteID != nil {
			updateQuery = updateQuery.Where("site_id = ?", *requiredSiteID)
		}
		return updateQuery.Update("zone_id", zoneID).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menetapkan CCTV ke zona",
		})
	}

	utils.LogAudit(config.DB, c, "ASSIGN_CCTV_ZONE", "ZONE",
		fmt.Sprintf("Menetapkan %d CCTV ke zona %s", len(ids), zone.Nama), &zoneID, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "CCTV berhasil ditugaskan ke zona",
		"data": fiber.Map{
			"zone_id":    zoneID,
			"cctv_count": len(ids),
		},
	})
}

// AssignUsersToZone menetapkan daftar teknisi ke sebuah zona (many-to-many via
// user_zones). Menimpa seluruh penugasan user pada zona tersebut. Admin saja.
// @Summary      Assign Teknisi ke Zona
// @Description  Menetapkan daftar teknisi ke sebuah zona. Seluruh penugasan user lama pada zona digantikan daftar baru (bila user_ids kosong, penugasan dibersihkan).
// @Tags         Zona
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Zona UUID"
// @Param        request body AssignUsersRequest true "Daftar ID User"
// @Success      200 {object} map[string]interface{} "Teknisi berhasil ditugaskan ke zona"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid atau ada user tidak ditemukan"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Zona tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menetapkan teknisi ke zona"
// @Router       /zones/{id}/assign-users [post]
func AssignUsersToZone(c *fiber.Ctx) error {
	zoneID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID zona tidak valid",
		})
	}

	var zone models.Zone
	if err := config.DB.First(&zone, "id = ?", zoneID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Zona tidak ditemukan",
		})
	}
	if !zoneMatchesActiveSite(c, zone) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: zona berada di site lain",
		})
	}

	var req AssignUsersRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}

	// Daftar user boleh kosong (berarti membersihkan seluruh penugasan zona).
	if len(req.UserIDs) > 0 {
		var found int64
		if err := config.DB.Model(&models.User{}).
			Where("id IN ? AND role = ? AND deleted_at IS NULL", req.UserIDs, "teknisi").
			Count(&found).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal memvalidasi daftar teknisi",
			})
		}
		if found != int64(len(req.UserIDs)) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Satu atau lebih teknisi pada daftar tidak ditemukan atau bukan berperan teknisi",
			})
		}
	}

	// Simpan daftar lama agar notifikasi hanya dikirim kepada teknisi yang baru
	// ditambahkan, bukan berulang kepada teknisi yang tetap berada di zona.
	var previousAssignments []models.UserZone
	if err := config.DB.Where("zone_id = ?", zoneID).Find(&previousAssignments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca penugasan zona sebelumnya",
		})
	}
	previousUserIDs := make(map[uuid.UUID]struct{}, len(previousAssignments))
	for _, assignment := range previousAssignments {
		previousUserIDs[assignment.UserID] = struct{}{}
	}

	// Timpa penugasan zona: hapus semua lalu sisipkan daftar baru.
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("zone_id = ?", zoneID).Delete(&models.UserZone{}).Error; err != nil {
			return err
		}
		for _, uid := range req.UserIDs {
			uz := models.UserZone{ZoneID: zoneID, UserID: uid}
			if err := tx.Create(&uz).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menetapkan teknisi ke zona",
		})
	}

	utils.LogAudit(config.DB, c, "ASSIGN_USER_ZONE", "ZONE",
		fmt.Sprintf("Menetapkan %d teknisi ke zona %s", len(req.UserIDs), zone.Nama), &zoneID, "", "")

	for _, userID := range req.UserIDs {
		if _, alreadyAssigned := previousUserIDs[userID]; alreadyAssigned {
			continue
		}
		if err := utils.NotifyUserSync(
			config.DB,
			userID,
			models.NotificationTypeZoneAssigned,
			"Penugasan zona baru",
			fmt.Sprintf("Anda ditugaskan ke zona %s. Buka Peta Sebaran CCTV untuk melihat cakupan tugas.", zone.Nama),
			"zone",
			&zoneID,
		); err != nil {
			utils.Log.Errorf("Gagal mengirim notifikasi penugasan zona %s ke user %s: %v", zoneID, userID, err)
		}
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Teknisi berhasil ditugaskan ke zona",
		"data": fiber.Map{
			"zone_id":    zoneID,
			"user_count": len(req.UserIDs),
		},
	})
}

// zoneMatchesActiveSite memastikan objek zona yang diakses cocok dengan site
// aktif yang ditetapkan middleware SiteContext atau header/cookie/query site.
func zoneMatchesActiveSite(c *fiber.Ctx, zone models.Zone) bool {
	activeSiteID, ok := resolveActiveSiteID(c)
	if !ok {
		return false
	}
	if activeSiteID == nil {
		return true
	}
	return zone.SiteID != nil && *zone.SiteID == *activeSiteID
}
