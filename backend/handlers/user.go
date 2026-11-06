package handlers

import (
	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UpdateUserRequest mendefinisikan skema validasi untuk memperbarui data user oleh Admin
type UpdateUserRequest struct {
	FullName string `json:"full_name" validate:"omitempty,min=3,max=100"`
	Email    string `json:"email" validate:"omitempty,email,max=100"`
	Role     string `json:"role" validate:"omitempty,oneof=admin teknisi"`
}

// ResetUserPasswordRequest mendefinisikan skema validasi untuk reset password paksa oleh Admin
type ResetUserPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=8,max=100,strongpassword"`
}

// GetAllUsers mendapatkan daftar seluruh pengguna dengan paginasi, pencarian, dan penyaringan
// @Summary      Mendapatkan Daftar Pengguna
// @Description  Mengembalikan daftar pengguna terdaftar berdasarkan parameter pencarian, pengurutan, dan paginasi. Hanya bisa diakses oleh Admin.
// @Tags         Manajemen Pengguna
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Nomor Halaman (default: 1)"
// @Param        limit query int false "Jumlah Data per Halaman (default: 10)"
// @Param        sort_by query string false "Kolom Pengurutan (username, full_name, email, role, created_at)"
// @Param        order query string false "Arah Pengurutan (asc, desc)"
// @Param        search query string false "Cari berdasarkan username, full_name, atau email"
// @Param        role query string false "Filter berdasarkan peran (admin, teknisi)"
// @Success      200 {object} map[string]interface{} "Sukses mendapatkan daftar pengguna"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil data"
// @Router       /users [get]
func GetAllUsers(c *fiber.Ctx) error {
	var users []models.User
	var totalRows int64

	limit, page, sort := utils.GetPaginationParams(c, []string{"username", "full_name", "email", "role", "created_at"})
	offset := (page - 1) * limit

	query := config.DB.Model(&models.User{})

	// Pencarian teks
	if search := c.Query("search"); search != "" {
		query = query.Where("username ILIKE ? OR full_name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Filter berdasarkan peran (mis. daftar teknisi untuk penugasan)
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}

	// Hitung total data terfilter
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total data pengguna",
		})
	}

	// Ambil data terpaginasi
	if err := query.Order(sort).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data pengguna",
		})
	}

	metadata := utils.CalculatePagination(totalRows, page, limit)

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       users,
			"pagination": metadata,
		},
	})
}

// GetUserByID mendapatkan detail informasi pengguna berdasarkan UUID
// @Summary      Mendapatkan Detail Pengguna
// @Description  Mengembalikan informasi lengkap satu pengguna berdasarkan UUID. Hanya bisa diakses oleh Admin.
// @Tags         Manajemen Pengguna
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User UUID"
// @Success      200 {object} map[string]interface{} "Sukses mendapatkan detail pengguna"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      404 {object} map[string]interface{} "Pengguna tidak ditemukan"
// @Router       /users/{id} [get]
func GetUserByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengguna tidak valid",
		})
	}

	var user models.User
	if err := config.DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Pengguna tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   user,
	})
}

// UpdateUser memperbarui data profil pengguna
// @Summary      Memperbarui Profil Pengguna
// @Description  Memperbarui data nama lengkap (full_name), email, atau peran (role) pengguna. Hanya bisa diakses oleh Admin.
// @Tags         Manajemen Pengguna
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User UUID"
// @Param        body body UpdateUserRequest true "Data pembaruan user"
// @Success      200 {object} map[string]interface{} "Sukses memperbarui pengguna"
// @Failure      400 {object} map[string]interface{} "Input tidak valid / Email duplikat"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      404 {object} map[string]interface{} "Pengguna tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui pengguna"
// @Router       /users/{id} [put]
func UpdateUser(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengguna tidak valid",
		})
	}

	var user models.User
	if err := config.DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Pengguna tidak ditemukan",
		})
	}

	var req UpdateUserRequest
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

	// Update bidang-bidang jika diisi
	roleChanged := false
	oldRole := user.Role
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		// Validasi email duplikat ke pengguna lain
		var duplicateUser models.User
		if err := config.DB.Where("email = ? AND id != ?", req.Email, user.ID).First(&duplicateUser).Error; err == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Email sudah terdaftar pada pengguna lain",
			})
		}
		user.Email = req.Email
	}
	if req.Role != "" && req.Role != user.Role {
		user.Role = req.Role
		roleChanged = true
	}

	// SECURITY: Jika role diubah, naikkan versi token agar seluruh sesi lama
	// (token JWT yang beredar) langsung dicabut dan user wajib login ulang.
	if roleChanged {
		user.TokenVersion++
	}

	// Jika peran diubah dari teknisi ke non-teknisi, rekonsiliasi tugas & jadwal aktif (Issue #184 / M11)
	if roleChanged && oldRole == "teknisi" && user.Role != "teknisi" {
		err = config.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
			return reconcileRevokedTechnicianGlobalAssignments(tx, user.ID)
		})
	} else {
		err = config.DB.Save(&user).Error
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui pengguna",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_USER", "USER", "Memperbarui profil pengguna: "+user.Username+" ("+user.Role+")", nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Pengguna berhasil diperbarui",
		"data":    user,
	})
}

// DeleteUser menghapus data pengguna secara soft-delete
// @Summary      Menghapus Pengguna (Soft Delete)
// @Description  Menghapus data pengguna dari sistem menggunakan mekanisme soft-delete. Hanya bisa diakses oleh Admin.
// @Tags         Manajemen Pengguna
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User UUID"
// @Success      200 {object} map[string]interface{} "Sukses menghapus pengguna"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid / Tidak bisa menghapus diri sendiri"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      404 {object} map[string]interface{} "Pengguna tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus pengguna"
// @Router       /users/{id} [delete]
func DeleteUser(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengguna tidak valid",
		})
	}

	// Cegah admin menghapus dirinya sendiri dari sistem
	currentUserID, ok := c.Locals("user_id").(string)
	if ok && currentUserID == id.String() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Anda tidak diperbolehkan menghapus akun sendiri yang sedang aktif",
		})
	}

	var user models.User
	if err := config.DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Pengguna tidak ditemukan",
		})
	}

	// Jika pengguna yang dihapus adalah teknisi, rekonsiliasi tugas & jadwal aktif (Issue #184 / M11)
	if user.Role == "teknisi" {
		err = config.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&user).Error; err != nil {
				return err
			}
			return reconcileRevokedTechnicianGlobalAssignments(tx, user.ID)
		})
	} else {
		err = config.DB.Delete(&user).Error
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus pengguna",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_USER", "USER", "Menghapus akun pengguna: "+user.Username+" ("+user.Role+")", nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Pengguna berhasil dihapus secara soft-delete",
	})
}

// ResetUserPassword menyetel ulang sandi masuk pengguna secara paksa oleh Admin
// @Summary      Reset Password Pengguna
// @Description  Menyetel ulang sandi pengguna secara paksa ke sandi baru pilihan Admin. Hanya bisa diakses oleh Admin.
// @Tags         Manajemen Pengguna
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User UUID"
// @Param        body body ResetUserPasswordRequest true "Data password baru"
// @Success      200 {object} map[string]interface{} "Sukses reset password"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid / Validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      404 {object} map[string]interface{} "Pengguna tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal enkripsi / Gagal menyimpan"
// @Router       /users/{id}/reset-password [post]
func ResetUserPassword(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengguna tidak valid",
		})
	}

	var user models.User
	if err := config.DB.First(&user, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Pengguna tidak ditemukan",
		})
	}

	var req ResetUserPasswordRequest
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

	// Enkripsi password baru
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengenkripsi password baru",
		})
	}

	user.PasswordHash = string(hashedPassword)

	// SECURITY: Naikkan versi token agar seluruh sesi lama pengguna langsung
	// dicabut setelah password di-reset.
	user.TokenVersion++

	// Paksa pengguna mengganti password pada login berikutnya.
	user.MustChangePassword = true

	if err := config.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan password baru ke database",
		})
	}

	utils.LogAudit(config.DB, c, "RESET_PASSWORD", "USER", "Admin melakukan reset password paksa untuk pengguna: "+user.Username, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Sandi pengguna berhasil disetel ulang",
	})
}
