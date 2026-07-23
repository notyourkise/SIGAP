package utils

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// PaginationMetadata menyimpan info halaman untuk respon JSON
type PaginationMetadata struct {
	TotalRows  int64 `json:"total_rows"`
	TotalPages int   `json:"total_pages"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
}

// GetPaginationParams mengambil parameter limit, page, dan sort dari query URL dengan whitelisting kolom
func GetPaginationParams(c *fiber.Ctx, allowedSortFields []string) (int, int, string) {
	limitStr := c.Query("limit", "10")
	pageStr := c.Query("page", "1")
	sortCol := c.Query("sort_by", "created_at")
	sortOrder := c.Query("order", "desc") // asc atau desc

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	// Validasi Whitelist kolom pengurutan (Mencegah SQL Injection di GORM .Order())
	isAllowed := false
	for _, field := range allowedSortFields {
		if sortCol == field {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		sortCol = "created_at"
	}

	// Validasi arah order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	sort := sortCol + " " + sortOrder
	return limit, page, sort
}

// CalculatePagination menghitung metadata halaman
func CalculatePagination(totalRows int64, page int, limit int) PaginationMetadata {
	totalPages := int(math.Ceil(float64(totalRows) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}
	return PaginationMetadata{
		TotalRows:  totalRows,
		TotalPages: totalPages,
		Page:       page,
		Limit:      limit,
	}
}
