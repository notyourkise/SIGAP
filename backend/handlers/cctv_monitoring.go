package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"sigap-backend/config"
	"sigap-backend/incidents"
	"sigap-backend/models"
	"sigap-backend/utils"
)

// enrichCCTVsWithPlannedOutage memperkaya daftar/detail CCTV dengan status jendela
// pemeliharaan terjadwal aktif ("Sedang pemeliharaan terjadwal" — Keputusan #1)
// tanpa mengganggu ekspektasi sqlmock lama ketika fitur monitoring belum diaktifkan.
func enrichCCTVsWithPlannedOutage(c *fiber.Ctx, cctvs []models.CCTV) {
	if config.DB == nil || len(cctvs) == 0 {
		return
	}
	if !IncidentMonitoringEnabled() && c.Query("include_planned_outage") != "true" {
		return
	}
	ids := make([]uuid.UUID, 0, len(cctvs))
	for i := range cctvs {
		ids = append(ids, cctvs[i].ID)
	}
	activeMap, err := (incidents.Service{DB: config.DB}).ActivePlannedOutageMap(c.UserContext(), ids, time.Now().UTC())
	if err != nil || len(activeMap) == 0 {
		return
	}
	for i := range cctvs {
		if info, ok := activeMap[cctvs[i].ID]; ok {
			outageID := info.OutageID
			endsAt := info.EndsAt
			cctvs[i].PlannedOutageActive = true
			cctvs[i].PlannedOutageID = &outageID
			cctvs[i].PlannedOutageReason = info.Reason
			cctvs[i].PlannedOutageEndsAt = &endsAt
		}
	}
}

// ReactivateAndVerifyCCTV menjalankan reaktivasi terverifikasi setelah unit CCTV baru dipasang
// menggantikan unit rusak yang sebelumnya dinonaktifkan lewat eskalasi penggantian (Keputusan #5):
//  1. Masuk melalui UnifiedNVRSessionGate yang sama dengan pemeriksaan terjadwal (mematuhi batas sesi
//     gabungan & batas GSM) sehingga tidak membuka stream baru saat gate penuh.
//  2. Setelah mendapat lease slot di dalam gate, mendaftarkan path stream ke MediaMTX dan menjalankan
//     verifikasi stream aktif (termasuk cek kemajuan byte RTP).
//  3. Hanya setelah stream terbukti live, mengaktifkan kembali status='aktif', monitoring_enabled=true,
//     dan stream_enabled=true. Jika menunggu melebihi batas waktu atau belum live, kamera tetap nonaktif.
func ReactivateAndVerifyCCTV(c *fiber.Ctx) error {
	if config.DB == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Database belum tersedia",
		})
	}
	if role, _ := c.Locals("role").(string); role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: hanya Admin yang diizinkan mengaktifkan kembali dan memverifikasi CCTV",
		})
	}

	cctv, ok := loadCCTVByID(c)
	if !ok {
		return nil
	}

	gate := getOrInitVerifierGate(config.DB)
	waitTimeout := resolveManualVerifyWaitTimeout()
	waitCtx, cancel := context.WithTimeout(c.UserContext(), waitTimeout)
	defer cancel()

	activeDB := config.DB
	var regErr error
	result, waitErr := gate.ExecuteInteractive(waitCtx, GateJob{
		CCTV:                  cctv,
		SkipCompletionHandler: true,
		JobProbe: func(_ context.Context, job GateJob) (streamVerificationResult, bool) {
			if err := registerStreamWithRetry(job.CCTV, 1); err != nil {
				regErr = err
				return streamVerificationResult{
					CCTV:       job.CCTV,
					State:      models.StreamStateUnknown,
					VerifiedAt: time.Now(),
					Error:      err.Error(),
				}, false
			}
			res := verifyCCTVStreamWithOptions(activeDB, job.CCTV, true, true, nil)
			if res.State != models.StreamStateLive && !job.CCTV.StreamEnabled {
				_ = streamClient.RemovePath(utils.BuildStreamPath(job.CCTV.ID.String()))
			}
			if !res.OpenedOnDemand {
				persistImmediateProbeTelemetry(activeDB, res)
			}
			return res, res.OpenedOnDemand
		},
	})
	if waitErr != nil {
		timedOut := errors.Is(waitErr, ErrGateWaitTimeout)
		msg := "Layanan antrean verifikasi stream sedang tidak tersedia; kamera tetap berstatus nonaktif"
		if timedOut {
			msg = fmt.Sprintf("Batas sesi verifikasi NVR/GSM sedang penuh (menunggu > %s); reaktivasi dibatalkan sebelum membuka stream dan kamera tetap nonaktif", waitTimeout.Round(time.Millisecond))
		}
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "error",
			"message": msg,
			"data": fiber.Map{
				"cctv_id":        cctv.ID.String(),
				"network_type":   networkType(cctv),
				"wait_timed_out": timedOut,
				"session_gate":   gate.Snapshot(),
			},
		})
	}

	if regErr != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mendaftarkan path RTSP ke MediaMTX sebelum reaktivasi; periksa URL/password RTSP kamera",
		})
	}

	if result.State != models.StreamStateLive {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"status":  "error",
			"message": "Stream kamera pengganti belum terbukti live (" + result.State + "). Kamera tetap berstatus nonaktif sampai stream berhasil diverifikasi.",
			"data": fiber.Map{
				"cctv_id":               cctv.ID.String(),
				"stream_verified_state": result.State,
				"stream_verify_error":   result.Error,
				"tracks":                result.Tracks,
				"byte_delta":            result.ByteDelta,
			},
		})
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":                 "aktif",
		"monitoring_enabled":     true,
		"stream_enabled":         true,
		"stream_verify_failures": 0,
		"stream_verified_state":  models.StreamStateLive,
		"stream_verified_at":     now,
		"stream_last_active_at":  now,
		"updated_at":             now,
	}
	if err := config.DB.WithContext(context.Background()).Model(&models.CCTV{}).Where("id = ?", cctv.ID).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengaktifkan kembali status kamera di database",
		})
	}

	_ = config.DB.First(&cctv, "id = ?", cctv.ID).Error
	utils.LogAudit(config.DB, c, "REACTIVATE_VERIFY_CCTV", "CCTV", "Reaktivasi & verifikasi stream berhasil untuk unit CCTV: "+cctv.Nama+" ("+cctv.KodeCCTV+")", nil, "", "")
	broadcastCCTVUpdate(cctv)
	sanitizeCCTVForResponse(&cctv)

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Stream terverifikasi live; CCTV kembali aktif dan masuk cakupan pemantauan",
		"data":    cctv,
	})
}
