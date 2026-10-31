package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// extractStreamCameraID mengekstrak streamPath ("cam-<uuid>") dan UUID CCTV
// dari URI/path permintaan seperti "/stream/cam-<uuid>/whep",
// "/hls/cam-<uuid>/index.m3u8", atau "cam-<uuid>".
func extractStreamCameraID(rawTarget string) (string, uuid.UUID, error) {
	cleaned := strings.TrimSpace(rawTarget)
	if cleaned == "" {
		return "", uuid.Nil, fmt.Errorf("path stream kosong")
	}
	if idx := strings.IndexByte(cleaned, '?'); idx >= 0 {
		cleaned = cleaned[:idx]
	}
	if strings.Contains(cleaned, "..") || strings.Contains(cleaned, `\`) {
		return "", uuid.Nil, fmt.Errorf("path stream tidak valid")
	}

	trimmed := strings.TrimPrefix(cleaned, "/")
	switch {
	case strings.HasPrefix(trimmed, "stream/"):
		trimmed = strings.TrimPrefix(trimmed, "stream/")
	case strings.HasPrefix(trimmed, "hls/"):
		trimmed = strings.TrimPrefix(trimmed, "hls/")
	}
	trimmed = strings.TrimPrefix(trimmed, "/")

	seg := strings.SplitN(trimmed, "/", 2)[0]
	if !strings.HasPrefix(seg, "cam-") {
		return "", uuid.Nil, fmt.Errorf("format path kamera tidak valid")
	}

	rawID := strings.TrimPrefix(seg, "cam-")
	camID, err := uuid.Parse(rawID)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("UUID kamera pada path tidak valid: %w", err)
	}

	return utils.BuildStreamPath(camID.String()), camID, nil
}

// authorizeStreamTarget memverifikasi bahwa target stream menuju kamera yang
// terdaftar, memiliki stream_enabled=true, dan berada dalam cakupan site aktif
// pemanggil (bila konteks site dipilih). Mengembalikan false bila respons
// penolakan sudah dikirim ke client.
func authorizeStreamTarget(c *fiber.Ctx, rawTarget string) (models.CCTV, string, bool) {
	streamPath, camID, err := extractStreamCameraID(rawTarget)
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Path stream CCTV tidak valid",
		})
		return models.CCTV{}, "", false
	}

	if config.DB == nil {
		_ = c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "error",
			"message": "Database belum tersedia",
		})
		return models.CCTV{}, "", false
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", camID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Data CCTV tidak ditemukan",
			})
			return models.CCTV{}, "", false
		}
		_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat data CCTV",
		})
		return models.CCTV{}, "", false
	}

	if !cctv.StreamEnabled {
		_ = c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Streaming belum diaktifkan",
		})
		return models.CCTV{}, "", false
	}

	if siteIDStr, ok := c.Locals("site_id").(string); ok && siteIDStr != "" {
		if cctv.SiteID == nil || cctv.SiteID.String() != siteIDStr {
			_ = c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: CCTV berada di luar site aktif Anda",
			})
			return models.CCTV{}, "", false
		}
	}

	return cctv, streamPath, true
}

// AuthorizeStreamAccess memvalidasi sesi Admin dan izin akses kamera untuk
// subrequest Nginx auth_request pada jalur /stream/* (WHEP) dan /hls/* (HLS).
func AuthorizeStreamAccess(c *fiber.Ctx) error {
	target := strings.TrimSpace(c.Get("X-Original-URI"))
	if target == "" {
		target = strings.TrimSpace(c.Query("uri"))
	}
	if target == "" {
		target = strings.TrimSpace(c.Query("path"))
	}
	if target == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Target stream wajib disertakan",
		})
	}

	cctv, streamPath, ok := authorizeStreamTarget(c, target)
	if !ok {
		return nil
	}

	c.Set("X-Stream-Path", streamPath)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"cctv_id": cctv.ID.String(),
			"path":    streamPath,
		},
	})
}

// ProxyWHEPStream meneruskan permintaan WebRTC WHEP (/stream/cam-<uuid>/whep*)
// ke MediaMTX setelah memvalidasi sesi Admin, status stream_enabled, dan
// cakupan site kamera.
func ProxyWHEPStream(c *fiber.Ctx) error {
	_, streamPath, ok := authorizeStreamTarget(c, c.Path())
	if !ok {
		return nil
	}

	rawSuffix := strings.TrimPrefix(c.Path(), "/stream")
	if strings.Contains(rawSuffix, "..") || strings.Contains(rawSuffix, `\`) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Jalur WHEP tidak valid",
		})
	}
	cleanSuffix := path.Clean("/" + strings.TrimPrefix(rawSuffix, "/"))
	expectedPrefix := "/" + streamPath + "/whep"
	if cleanSuffix != expectedPrefix && !strings.HasPrefix(cleanSuffix, expectedPrefix+"/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Jalur WHEP tidak valid",
		})
	}

	baseURL := ""
	if streamClient != nil {
		baseURL = streamClient.WebRTCBaseURL()
	}
	if baseURL == "" {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Server streaming belum dikonfigurasi",
		})
	}

	upstreamURL := baseURL + cleanSuffix
	if rawQuery := string(c.Request().URI().QueryString()); rawQuery != "" {
		upstreamURL += "?" + rawQuery
	}

	req, err := http.NewRequest(c.Method(), upstreamURL, bytes.NewReader(c.Body()))
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyiapkan permintaan ke server streaming",
		})
	}
	if ct := c.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if accept := c.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}
	if ifMatch := c.Get("If-Match"); ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	if streamClient != nil && streamClient.HTTPClient != nil {
		httpClient = streamClient.HTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghubungi server streaming",
		})
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca respons server streaming",
		})
	}

	copyStreamResponseHeaders(c, resp.Header, true)
	return c.Status(resp.StatusCode).Send(body)
}

// ProxyHLSStream meneruskan permintaan playlist/segmen HLS (/hls/cam-<uuid>/*)
// ke MediaMTX setelah memvalidasi sesi Admin, status stream_enabled, dan
// cakupan site kamera.
func ProxyHLSStream(c *fiber.Ctx) error {
	switch c.Method() {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
	default:
		return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
			"status":  "error",
			"message": "Metode HTTP tidak diizinkan untuk HLS",
		})
	}

	_, streamPath, ok := authorizeStreamTarget(c, c.Path())
	if !ok {
		return nil
	}

	rawSuffix := strings.TrimPrefix(c.Path(), "/hls")
	if strings.Contains(rawSuffix, "..") || strings.Contains(rawSuffix, `\`) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Jalur HLS tidak valid",
		})
	}
	cleanSuffix := path.Clean("/" + strings.TrimPrefix(rawSuffix, "/"))
	expectedPrefix := "/" + streamPath + "/"
	if !strings.HasPrefix(cleanSuffix, expectedPrefix) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Jalur HLS tidak valid",
		})
	}

	baseURL := ""
	if streamClient != nil {
		baseURL = streamClient.HLSBaseURL()
	}
	if baseURL == "" {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Server HLS belum dikonfigurasi",
		})
	}

	upstreamURL := baseURL + cleanSuffix
	if rawQuery := string(c.Request().URI().QueryString()); rawQuery != "" {
		upstreamURL += "?" + rawQuery
	}

	req, err := http.NewRequest(c.Method(), upstreamURL, nil)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyiapkan permintaan HLS",
		})
	}
	if accept := c.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	if streamClient != nil && streamClient.HTTPClient != nil {
		httpClient = streamClient.HTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghubungi server HLS",
		})
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca respons server HLS",
		})
	}

	copyStreamResponseHeaders(c, resp.Header, false)
	return c.Status(resp.StatusCode).Send(body)
}

func copyStreamResponseHeaders(c *fiber.Ctx, hdr http.Header, rewriteStreamLocation bool) {
	allowed := []string{
		"Content-Type",
		"Cache-Control",
		"ETag",
		"Accept-Patch",
		"Access-Control-Expose-Headers",
		"Link",
	}
	for _, key := range allowed {
		if val := hdr.Get(key); val != "" {
			c.Set(key, val)
		}
	}
	if loc := hdr.Get("Location"); loc != "" {
		if rewriteStreamLocation && strings.HasPrefix(loc, "/cam-") {
			loc = "/stream" + loc
		}
		c.Set("Location", loc)
	}
}
