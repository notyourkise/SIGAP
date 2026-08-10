package handlers

import (
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=4,max=50"`
	Password string `json:"password" validate:"required,min=8,max=100,strongpassword"`
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Role     string `json:"role" validate:"required,oneof=admin teknisi"`
}

// Login memproses permintaan login pengguna dan memberikan token JWT jika sukses
// @Summary      Login Pengguna
// @Description  Masuk ke sistem SIGAP menggunakan username/email dan password, mengembalikan token JWT.
// @Tags         Autentikasi
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Kredensial Login"
// @Success      200 {object} map[string]interface{} "Sukses login dengan token JWT"
// @Failure      400 {object} map[string]interface{} "Format input tidak valid atau data kurang"
// @Failure      401 {object} map[string]interface{} "Username atau password salah"
// @Router       /auth/login [post]
func Login(c *fiber.Ctx) error {
	var req LoginRequest
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

	var user models.User
	// Mendukung login menggunakan username ataupun email
	err := config.DB.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error
	if err != nil {
		utils.LogAudit(config.DB, c, "LOGIN_FAILED", "AUTH", "Percobaan login gagal untuk username/email: "+req.Username, nil, req.Username, "guest")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Username atau password salah",
		})
	}

	// Bandingkan password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		utils.LogAudit(config.DB, c, "LOGIN_FAILED", "AUTH", "Password salah untuk username: "+user.Username, &user.ID, user.Username, user.Role)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Username atau password salah",
		})
	}

	// Generate JWT Token â€” sertakan versi token agar token lama bisa dicabut
	token, err := utils.GenerateToken(user.ID.String(), user.Role, user.TokenVersion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghasilkan token otentikasi",
		})
	}

	// Simpan token di cookie httpOnly (tidak dapat dibaca JavaScript) + token CSRF
	// pola double-submit. Token JWT tidak lagi dikembalikan di body respons.
	csrfToken := utils.GenerateCSRFToken()
	// Bersihkan cookie lama pada semua path yang pernah dipakai sebelum menerbitkan
	// sesi baru agar browser tidak memilih token stale untuk request /api/*.
	utils.ClearSessionCookies(c)
	utils.SetSessionCookie(c, token)
	utils.SetCSRFCookie(c, csrfToken)

	// Catat audit log login berhasil
	utils.LogAudit(config.DB, c, "LOGIN_SUCCESS", "AUTH", "Pengguna berhasil login ke sistem", &user.ID, user.Username, user.Role)

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"csrf_token": csrfToken,
			"user": fiber.Map{
				"id":                   user.ID,
				"username":             user.Username,
				"full_name":            user.FullName,
				"email":                user.Email,
				"role":                 user.Role,
				"must_change_password": user.MustChangePassword,
			},
		},
	})
}

// ChangePasswordRequest adalah body untuk mengganti password sendiri (self-service).
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"omitempty"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=100,strongpassword"`
}

// ChangePassword mengubah password pengguna yang sedang login (self-service).
// @Summary      Ganti Password Sendiri
// @Description  Mengganti password akun sendiri (self-service). Wajib menyertakan password lama.
//
//	Setelah sukses, seluruh sesi lama dicabut dan sesi saat ini dilanjutkan dengan token baru.
//
// @Tags         Autentikasi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body ChangePasswordRequest true "Password lama & baru"
// @Success      200 {object} map[string]interface{} "Password berhasil diubah, sesi baru diterbitkan"
// @Failure      400 {object} map[string]interface{} "Validasi input gagal / password lama salah / password baru sama"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan password baru"
// @Router       /auth/change-password [post]
func ChangePassword(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak ditemukan dalam token",
		})
	}

	var req ChangePasswordRequest
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

	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Pengguna tidak ditemukan",
		})
	}

	// Verifikasi password lama bila disertakan. Jika kosong (mode tanpa password
	// lama), verifikasi dilewati.
	if req.OldPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
			utils.LogAudit(config.DB, c, "CHANGE_PASSWORD_FAILED", "AUTH", "Gagal mengganti password: password lama salah (username: "+user.Username+")", &user.ID, user.Username, user.Role)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Password lama tidak sesuai",
			})
		}
	}

	// Cegah memakai password yang sama dengan sebelumnya (bila password lama disertakan)
	if req.OldPassword != "" && req.OldPassword == req.NewPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Password baru harus berbeda dari password lama",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengenkripsi password baru",
		})
	}

	// Simpan password baru, tandai sudah tidak perlu ganti password paksa, dan
	// naikkan versi token agar seluruh sesi lama (di perangkat lain) dicabut.
	user.PasswordHash = string(hashedPassword)
	user.MustChangePassword = false
	user.TokenVersion++
	if err := config.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan password baru ke database",
		})
	}

	// Terbitkan token baru (versi terbaru) & perbarui cookie sesi agar sesi ini
	// tetap aktif sementara semua sesi lain dicabut.
	token, err := utils.GenerateToken(user.ID.String(), user.Role, user.TokenVersion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghasilkan token sesi baru",
		})
	}
	csrfToken := utils.GenerateCSRFToken()
	utils.SetSessionCookie(c, token)
	utils.SetCSRFCookie(c, csrfToken)

	utils.LogAudit(config.DB, c, "CHANGE_PASSWORD", "AUTH", "Pengguna mengganti password sendiri (username: "+user.Username+")", &user.ID, user.Username, user.Role)

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"csrf_token": csrfToken,
			"user": fiber.Map{
				"id":                   user.ID,
				"username":             user.Username,
				"full_name":            user.FullName,
				"email":                user.Email,
				"role":                 user.Role,
				"must_change_password": user.MustChangePassword,
			},
		},
	})
}

// Logout menginvalidasi token JWT yang sedang aktif agar tidak bisa digunakan lagi.
// Endpoint ini idempotent: token kosong atau invalid tetap menghasilkan respons sukses
// dan selalu menghapus cookie sesi/CSRF dari browser.
// @Summary      Logout Pengguna
// @Description  Merevoke token JWT aktif bila valid dan selalu menghapus cookie sesi/CSRF. Token invalid atau kosong tetap dianggap logout berhasil.
// @Tags         Autentikasi
// @Produce      json
// @Success      200 {object} map[string]interface{} "Logout berhasil, token diinvalidasi"
// @Router       /auth/logout [post]
func Logout(c *fiber.Ctx) error {
	tokenString := utils.ExtractTokenString(c)
	if tokenString == "" {
		utils.ClearSessionCookies(c)
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Logout berhasil",
		})
	}

	// Parse token untuk mendapatkan waktu expiry
	parsedToken, err := utils.ValidateToken(tokenString)
	if err != nil || !parsedToken.Valid {
		// Token sudah tidak valid, anggap logout berhasil
		utils.ClearSessionCookies(c)
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Logout berhasil",
		})
	}

	// Ambil expiry dari claims untuk cleanup otomatis blacklist
	expiry := time.Now().Add(25 * time.Hour) // default fallback
	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			expiry = time.Unix(int64(exp), 0)
		}
	}

	// Masukkan token ke blacklist â€” token tidak akan bisa digunakan lagi
	utils.RevokeToken(tokenString, expiry)
	utils.ClearSessionCookies(c)

	// Catat audit log logout
	if userIDStr, ok := c.Locals("user_id").(string); ok && userIDStr != "" {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			role, _ := c.Locals("role").(string)
			utils.LogAudit(config.DB, c, "LOGOUT", "AUTH", "Pengguna melakukan logout dari sistem", &uid, "", role)
		}
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Logout berhasil. Token telah diinvalidasi.",
	})
}

// GetMyProfile mengembalikan data profil pengguna yang sedang login
// @Summary      Profil Pengguna Saat Ini
// @Description  Mengembalikan data profil lengkap pengguna yang sedang terautentikasi berdasarkan token JWT.
// @Tags         Autentikasi
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Data profil pengguna"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      404 {object} map[string]interface{} "Pengguna tidak ditemukan"
// @Router       /auth/me [get]
func GetMyProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak ditemukan dalam token",
		})
	}

	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Pengguna tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"id":                   user.ID,
			"username":             user.Username,
			"full_name":            user.FullName,
			"email":                user.Email,
			"role":                 user.Role,
			"created_at":           user.CreatedAt,
			"must_change_password": user.MustChangePassword,
			"profile_photo_url":    user.ProfilePhotoURL,
			"email_notifications":  user.EmailNotifications,
			"email_daily_digest":   user.EmailDailyDigest,
			"email_overdue":        user.EmailOverdue,
			"email_stock_low":      user.EmailStockLow,
		},
	})
}

// UpdateMyProfileRequest memuat field yang boleh diubah oleh pengguna untuk dirinya sendiri.
type UpdateMyProfileRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	// ProfilePhotoURL bersifat opsional: nil = biarkan foto saat ini, "" = hapus
	// foto, selain itu harus path file upload SIGAP ("/api/uploads/...").
	ProfilePhotoURL *string `json:"profile_photo_url"`
}

// UpdateMyProfile memperbarui profil pengguna yang sedang login (nama tampilan & foto profil).
// @Summary      Perbarui Profil Sendiri
// @Description  Mengubah nama tampilan (full_name) dan/atau foto profil (profile_photo_url, path dari endpoint upload) milik pengguna yang sedang login.
// @Tags         Autentikasi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body UpdateMyProfileRequest true "Data Profil yang diperbarui"
// @Success      200 {object} map[string]interface{} "Profil berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "Format/validasi tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Pengguna tidak ditemukan"
// @Router       /auth/profile [put]
func UpdateMyProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak ditemukan dalam token",
		})
	}

	var req UpdateMyProfileRequest
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

	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Pengguna tidak ditemukan",
		})
	}

	user.FullName = strings.TrimSpace(req.FullName)
	if req.ProfilePhotoURL != nil {
		photo := strings.TrimSpace(*req.ProfilePhotoURL)
		if photo != "" && !isSafeUploadPath(photo) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Path foto profil tidak valid",
			})
		}
		user.ProfilePhotoURL = photo
	}
	if err := config.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui profil",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_PROFILE", "USER", "Pengguna memperbarui profilnya", &user.ID, "", user.Role)

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"id":                user.ID,
			"username":          user.Username,
			"full_name":         user.FullName,
			"email":             user.Email,
			"role":              user.Role,
			"profile_photo_url": user.ProfilePhotoURL,
		},
	})
}

// isSafeUploadPath memastikan path hanya menunjuk ke file upload SIGAP yang aman.
func isSafeUploadPath(path string) bool {
	if !strings.HasPrefix(path, "/api/uploads/") {
		return false
	}
	rest := strings.TrimPrefix(path, "/api/uploads/")
	if rest == "" || strings.Contains(rest, "..") || strings.Contains(rest, "/") || strings.Contains(rest, "\\") {
		return false
	}
	return true
}

// CreateUser digunakan oleh Admin untuk meregistrasikan akun baru (Admin/Teknisi)
// @Summary      Registrasi User Baru
// @Description  Membuat akun user baru (Admin/Teknisi). Hanya dapat diakses oleh Admin.
// @Tags         Autentikasi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateUserRequest true "Data User Baru"
// @Success      201 {object} map[string]interface{} "User baru berhasil dibuat"
// @Failure      400 {object} map[string]interface{} "Format tidak valid, email/username sudah dipakai, role tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Router       /auth/users [post]
func CreateUser(c *fiber.Ctx) error {
	var req CreateUserRequest
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

	// Periksa apakah username atau email sudah terdaftar
	var existingUser models.User
	err := config.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Username atau Email sudah terdaftar",
		})
	}

	// Enkripsi password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengenkripsi password",
		})
	}

	newUser := models.User{
		Username:           req.Username,
		PasswordHash:       string(hashedPassword),
		FullName:           req.FullName,
		Email:              req.Email,
		Role:               req.Role,
		MustChangePassword: true, // user baru wajib ganti password saat login pertama
	}

	if err := config.DB.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan user baru ke database",
		})
	}

	// Catat audit log pemuatan user baru oleh Admin
	if adminIDStr, ok := c.Locals("user_id").(string); ok && adminIDStr != "" {
		if adminID, err := uuid.Parse(adminIDStr); err == nil {
			adminRole, _ := c.Locals("role").(string)
			utils.LogAudit(config.DB, c, "CREATE_USER", "USER", "Admin membuat pengguna baru: "+newUser.Username+" ("+newUser.Role+")", &adminID, "", adminRole)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"id":        newUser.ID,
			"username":  newUser.Username,
			"full_name": newUser.FullName,
			"email":     newUser.Email,
			"role":      newUser.Role,
		},
	})
}
