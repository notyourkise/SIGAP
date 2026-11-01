package utils

import (
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// DefaultAppTimezone adalah zona waktu operasional default SIGAP (WITA).
const DefaultAppTimezone = "Asia/Makassar"

var appLocationCache sync.Map

// AppLocation mengembalikan zona waktu operasional aplikasi dari APP_TIMEZONE.
// Nilai kosong memakai Asia/Makassar. Nama zona yang tidak valid atau tidak
// tersedia di image runtime akan jatuh ke fixed zone WITA (UTC+8) agar endpoint
// yang menghitung batas hari maupun jadwal pemeliharaan tetap konsisten.
func AppLocation() *time.Location {
	name := strings.TrimSpace(os.Getenv("APP_TIMEZONE"))
	if name == "" {
		name = DefaultAppTimezone
	}

	if cached, ok := appLocationCache.Load(name); ok {
		return cached.(*time.Location)
	}

	location, err := time.LoadLocation(name)
	if err != nil {
		location = time.FixedZone("WITA", 8*60*60)
	}
	appLocationCache.Store(name, location)
	return location
}

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

	// Batas maksimal jumlah data per halaman untuk mencegah dump seluruh tabel / DoS memori.
	const maxLimit = 100

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}
	// Batasi nomor halaman agar offset tidak membesar tak terkendali.
	const maxPage = 1000000
	if page > maxPage {
		page = maxPage
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

// ParseDateRangeParams membaca filter rentang tanggal (start_date & end_date,
// format YYYY-MM-DD) dari query URL menggunakan zona waktu operasional aplikasi
// (default WITA / Asia/Makassar). Mengembalikan batas awal dan akhir waktu;
// batas akhir di-increment satu hari agar mencakup seluruh tanggal tersebut.
// Nilai nil menandakan parameter tidak disediakan / format tidak valid.
func ParseDateRangeParams(c *fiber.Ctx) (*time.Time, *time.Time) {
	loc := AppLocation()
	var start, end *time.Time

	if startStr := strings.TrimSpace(c.Query("start_date")); startStr != "" {
		if t, err := parseDateFilterBound(startStr, loc); err == nil {
			start = &t
		}
	}
	if endStr := strings.TrimSpace(c.Query("end_date")); endStr != "" {
		if t, err := parseDateFilterBound(endStr, loc); err == nil {
			t = t.AddDate(0, 0, 1)
			end = &t
		}
	}

	return start, end
}

func parseDateFilterBound(raw string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = AppLocation()
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.In(loc), nil
	}
	return time.ParseInLocation("2006-01-02", raw, loc)
}
