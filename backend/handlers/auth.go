package handlers

import (
	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=4,max=50"`
	Password string `json:"password" validate:"required,min=6"`
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Username atau password salah",
		})
	}

	// Bandingkan password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Username atau password salah",
		})
	}

	// Generate JWT Token
	token, err := utils.GenerateToken(user.ID.String(), user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghasilkan token otentikasi",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"token": token,
			"user": fiber.Map{
				"id":        user.ID,
				"username":  user.Username,
				"full_name": user.FullName,
				"email":     user.Email,
				"role":      user.Role,
			},
		},
	})
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
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		Email:        req.Email,
		Role:         req.Role,
	}

	if err := config.DB.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan user baru ke database",
		})
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
