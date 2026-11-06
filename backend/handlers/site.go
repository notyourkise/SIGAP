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

// CreateSiteRequest adalah body pembuatan site baru (Admin saja).
type CreateSiteRequest struct {
	Nama         string   `json:"nama" validate:"required,min=2,max=100"`
	Kode         string   `json:"kode" validate:"required,min=1,max=20"`
	Alamat       string   `json:"alamat" validate:"omitempty"`
	KoordinatLat *float64 `json:"koordinat_lat" validate:"omitempty"`
	KoordinatLng *float64 `json:"koordinat_lng" validate:"omitempty"`
	IsDefault    bool     `json:"is_default" validate:"omitempty"`
}

// UpdateSiteRequest adalah body pembaruan site (Admin saja).
type UpdateSiteRequest struct {
	Nama         string   `json:"nama" validate:"omitempty,min=2,max=100"`
	Kode         string   `json:"kode" validate:"omitempty,min=1,max=20"`
	Alamat       string   `json:"alamat" validate:"omitempty"`
	KoordinatLat *float64 `json:"koordinat_lat" validate:"omitempty"`
	KoordinatLng *float64 `json:"koordinat_lng" validate:"omitempty"`
	IsDefault    bool     `json:"is_default" validate:"omitempty"`
}

// AssignSiteUsersRequest adalah body assign user ke sebuah site (Admin saja).
type AssignSiteUsersRequest struct {
	// UserIDs adalah daftar ID user (teknisi) yang akan ditetapkan ke site.
	UserIDs []uuid.UUID `json:"user_ids" validate:"required"`
}

// SiteListRow adalah item daftar site dengan jumlah user.
type SiteListRow struct {
	models.Site
	UserCount int64 `gorm:"column:user_count;->" json:"user_count"`
}

// applySiteMembershipScope membatasi daftar site non-admin ke site yang
// ditugaskan kepadanya. Admin tetap melihat seluruh site. Scope ini berbeda
// dari site aktif: daftar ini dipakai SiteSwitcher agar user multi-site tetap
// dapat berpindah kawasan.
func applySiteMembershipScope(c *fiber.Ctx, query *gorm.DB) *gorm.DB {
	role, _ := c.Locals("role").(string)
	if role == "" || role == "admin" {
		return query
	}

	userIDStr, _ := c.Locals("user_id").(string)
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		return query.Where("1 = 0")
	}

	return query.Where(
		"EXISTS (SELECT 1 FROM user_sites WHERE user_sites.site_id = sites.id AND user_sites.user_id = ?)",
		uid,
	)
}

// GetAllSites mengambil daftar site terpaginasi (semua role login) beserta
// jumlah user per site. Non-admin hanya menerima site yang ditugaskan.
// @Summary      Ambil Semua Site
// @Description  Mendapatkan daftar kawasan operasional (site) terpaginasi dengan pencarian nama dan jumlah user per site. Non-admin hanya melihat site yang ditugaskan kepadanya.
// @Tags         Site
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        search query string false "Pencarian berdasarkan nama site"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar site terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: belum ditugaskan ke site"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar site"
// @Router       /sites [get]
func GetAllSites(c *fiber.Ctx) error {
	var rows []SiteListRow
	search := strings.TrimSpace(c.Query("search"))
	countQuery := applySiteMembershipScope(c, config.DB.Model(&models.Site{}))

	if search != "" {
		like := "%" + search + "%"
		countQuery = countQuery.Where("nama ILIKE ? OR kode ILIKE ?", like, like)
	}

	var totalRows int64
	if err := countQuery.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total site",
		})
	}

	limit, page, sort := sitePaginationParams(c)
	offset := (page - 1) * limit
	query := applySiteMembershipScope(c, config.DB.Model(&models.Site{})).
		Select(`sites.*, (SELECT COUNT(*) FROM user_sites WHERE user_sites.site_id = sites.id) AS user_count`)
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("sites.nama ILIKE ? OR sites.kode ILIKE ?", like, like)
	}
	if err := query.Offset(offset).Limit(limit).Order(sort).Scan(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar site",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       rows,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// sitePaginationParams memvalidasi sorting sebelum dipakai pada ORDER BY.
// Alias user_count berasal dari subquery SELECT di GetAllSites.
func sitePaginationParams(c *fiber.Ctx) (limit, page int, sort string) {
	limit, page, _ = utils.GetPaginationParams(c, []string{"id", "nama", "kode", "alamat", "created_at", "updated_at"})
	sortBy := c.Query("sort_by", "nama")
	order := c.Query("order", "asc")
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	columns := map[string]string{
		"id":         "sites.id",
		"nama":       "sites.nama",
		"kode":       "sites.kode",
		"alamat":     "sites.alamat",
		"teknisi":    "user_count",
		"user_count": "user_count",
		"created_at": "sites.created_at",
		"updated_at": "sites.updated_at",
	}
	column, ok := columns[sortBy]
	if !ok {
		column = "sites.nama"
	}
	return limit, page, column + " " + order
}

// GetAllSitesForSelect mengambil site tanpa paginasi untuk dropdown filter.
// Dapat diakses semua role; non-admin hanya menerima site yang ditugaskan.
// @Summary      Ambil Semua Site (Select)
// @Description  Mendapatkan site (id + nama + kode) tanpa paginasi untuk keperluan dropdown/filter. Non-admin hanya melihat site yang ditugaskan kepadanya.
// @Tags         Site
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Daftar site (tanpa paginasi)"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: belum ditugaskan ke site"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar site"
// @Router       /sites/all [get]
func GetAllSitesForSelect(c *fiber.Ctx) error {
	var items []models.Site
	query := applySiteMembershipScope(c, config.DB.Model(&models.Site{}))
	if err := query.Order("nama asc").Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar site",
		})
	}
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   items,
	})
}

// GetSiteByID mengambil detail satu site. Non-admin hanya dapat membuka site
// yang ditugaskan; site lain diperlakukan sebagai tidak ditemukan.
// @Summary      Ambil Site Detail
// @Description  Mendapatkan detail satu site beserta user yang ditugaskan. Non-admin hanya dapat membuka site yang ditugaskan kepadanya.
// @Tags         Site
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Site UUID"
// @Success      200 {object} map[string]interface{} "Data site ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Site tidak ditemukan atau bukan penugasan pengguna"
// @Router       /sites/{id} [get]
func GetSiteByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID site tidak valid",
		})
	}

	var item models.Site
	query := applySiteMembershipScope(c, config.DB.Preload("Users").Where("id = ?", id))
	if err := query.First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Site tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// CreateSite membuat site baru (Admin saja).
// @Summary      Tambah Site
// @Description  Membuat kawasan operasional (site) baru.
// @Tags         Site
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateSiteRequest true "Data Site Baru"
// @Success      201 {object} map[string]interface{} "Site berhasil disimpan"
// @Failure      400 {object} map[string]interface{} "Format input salah, validasi gagal, atau kode duplikat"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan site ke database"
// @Router       /sites [post]
func CreateSite(c *fiber.Ctx) error {
	var req CreateSiteRequest
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

	// Cek keunikan kode (termasuk yang di-soft-delete)
	var existing models.Site
	if err := config.DB.Unscoped().Where("kode = ?", strings.TrimSpace(req.Kode)).First(&existing).Error; err == nil {
		if existing.DeletedAt.Valid {
			// Hapus permanen data soft-deleted agar tidak memicu unique constraint.
			if err := config.DB.Unscoped().Delete(&existing).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "Gagal membersihkan data site usang di database",
				})
			}
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Kode site sudah digunakan oleh site lain",
			})
		}
	}

	item := models.Site{
		Nama:         strings.TrimSpace(req.Nama),
		Kode:         strings.TrimSpace(req.Kode),
		Alamat:       req.Alamat,
		KoordinatLat: req.KoordinatLat,
		KoordinatLng: req.KoordinatLng,
		IsDefault:    req.IsDefault,
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan site ke database",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_SITE", "SITE",
		"Membuat site baru: "+item.Nama, nil, "", "")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// UpdateSite memperbarui data site (Admin saja).
// @Summary      Update Site
// @Description  Memperbarui nama, kode, alamat, koordinat, atau status default site.
// @Tags         Site
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Site UUID"
// @Param        request body UpdateSiteRequest true "Data Update Site"
// @Success      200 {object} map[string]interface{} "Site berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, validasi gagal, atau kode duplikat"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Site tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui site"
// @Router       /sites/{id} [put]
func UpdateSite(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID site tidak valid",
		})
	}

	var item models.Site
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Site tidak ditemukan",
		})
	}

	var req UpdateSiteRequest
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

	// Cek keunikan kode bila diubah (abaikan site itu sendiri).
	if req.Kode != "" && strings.TrimSpace(req.Kode) != item.Kode {
		var dup models.Site
		if err := config.DB.Unscoped().Where("kode = ?", strings.TrimSpace(req.Kode)).First(&dup).Error; err == nil {
			if !dup.DeletedAt.Valid && dup.ID != item.ID {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "Kode site sudah digunakan oleh site lain",
				})
			}
		}
	}

	if req.Nama != "" {
		item.Nama = strings.TrimSpace(req.Nama)
	}
	if req.Kode != "" {
		item.Kode = strings.TrimSpace(req.Kode)
	}
	if req.Alamat != "" {
		item.Alamat = req.Alamat
	}
	if req.KoordinatLat != nil {
		item.KoordinatLat = req.KoordinatLat
	}
	if req.KoordinatLng != nil {
		item.KoordinatLng = req.KoordinatLng
	}
	item.IsDefault = req.IsDefault

	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui site",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_SITE", "SITE",
		"Memperbarui site: "+item.Nama, nil, "", "")

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// DeleteSite menghapus site secara soft-delete (Admin saja). Site default
// tidak dapat dihapus.
// @Summary      Hapus Site
// @Description  Menghapus site (soft-delete). Site default tidak dapat dihapus.
// @Tags         Site
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Site UUID"
// @Success      200 {object} map[string]interface{} "Site berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid atau site adalah site default"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Site tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus site"
// @Router       /sites/{id} [delete]
func DeleteSite(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID site tidak valid",
		})
	}

	var item models.Site
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Site tidak ditemukan",
		})
	}

	// Cegah penghapusan site default.
	if item.IsDefault {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Tidak dapat menghapus site default",
		})
	}

	if err := config.DB.Delete(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus site",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_SITE", "SITE",
		"Menghapus site: "+item.Nama, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Site berhasil dihapus",
	})
}

// AssignUsersToSite menetapkan daftar teknisi ke sebuah site (many-to-many via
// user_sites). Menimpa seluruh penugasan user pada site tersebut. Admin saja.
// @Summary      Assign Teknisi ke Site
// @Description  Menetapkan daftar teknisi ke sebuah site. Seluruh penugasan user lama pada site digantikan daftar baru (bila user_ids kosong, penugasan dibersihkan).
// @Tags         Site
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Site UUID"
// @Param        request body AssignSiteUsersRequest true "Daftar ID User"
// @Success      200 {object} map[string]interface{} "Teknisi berhasil ditugaskan ke site"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid atau ada user tidak ditemukan"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Site tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menetapkan teknisi ke site"
// @Router       /sites/{id}/assign-users [post]
func AssignUsersToSite(c *fiber.Ctx) error {
	siteID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID site tidak valid",
		})
	}

	var site models.Site
	if err := config.DB.First(&site, "id = ?", siteID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Site tidak ditemukan",
		})
	}

	var req AssignSiteUsersRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}

	// Daftar user boleh kosong (berarti membersihkan seluruh penugasan site).
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

	// Timpa penugasan site: hapus semua lalu sisipkan daftar baru.
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id = ?", siteID).Delete(&models.UserSite{}).Error; err != nil {
			return err
		}
		for _, uid := range req.UserIDs {
			us := models.UserSite{SiteID: siteID, UserID: uid}
			if err := tx.Create(&us).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menetapkan teknisi ke site",
		})
	}

	utils.LogAudit(config.DB, c, "ASSIGN_USER_SITE", "SITE",
		fmt.Sprintf("Menetapkan %d teknisi ke site %s (%s)", len(req.UserIDs), site.Nama, siteID.String()), nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Teknisi berhasil ditugaskan ke site",
		"data": fiber.Map{
			"site_id":    siteID,
			"user_count": len(req.UserIDs),
		},
	})
}
