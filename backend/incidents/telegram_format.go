package incidents

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"sigap-backend/models"
)

var witaZone = time.FixedZone("WITA", 8*3600)

// TelegramCoalesceWindow mengembalikan jendela penahanan penggabungan pesan (coalesce)
// dari env CCTV_TELEGRAM_COALESCE_SECONDS (0..300 detik), fallback ke DefaultCoalesceWindow.
func TelegramCoalesceWindow() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CCTV_TELEGRAM_COALESCE_SECONDS"))
	if raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs >= 0 && secs <= 300 {
			return time.Duration(secs) * time.Second
		}
	}
	return DefaultCoalesceWindow
}

func formatWITA(t time.Time) string {
	return t.In(witaZone).Format("15:04:05") + " WITA"
}

func cleanText(s string) string {
	return strings.TrimSpace(s)
}

func formatSingleCameraLabel(cam models.CCTV) string {
	code := cleanText(cam.KodeCCTV)
	name := cleanText(cam.Nama)
	loc := cleanText(cam.LokasiArea)

	base := name
	if code != "" && name != "" {
		base = fmt.Sprintf("%s - %s", code, name)
	} else if code != "" {
		base = code
	}
	if loc != "" {
		return fmt.Sprintf("%s (%s)", base, loc)
	}
	return base
}

func formatMembersSummary(members []Member, fallbackTitle string) string {
	if len(members) == 0 {
		return cleanText(fallbackTitle)
	}
	active := make([]Member, 0, len(members))
	for _, m := range members {
		if m.IsActiveMember {
			active = append(active, m)
		}
	}
	target := active
	if len(target) == 0 {
		target = members
	}
	if len(target) == 1 {
		m := target[0]
		code := cleanText(m.CameraCode)
		name := cleanText(m.CameraName)
		loc := cleanText(m.LocationArea)
		base := name
		if code != "" && name != "" {
			base = fmt.Sprintf("%s - %s", code, name)
		} else if code != "" {
			base = code
		}
		if loc != "" {
			return fmt.Sprintf("%s (%s)", base, loc)
		}
		return base
	}

	preview := make([]string, 0, 3)
	for idx, m := range target {
		if idx >= 3 {
			break
		}
		lbl := cleanText(m.CameraCode)
		if lbl == "" {
			lbl = cleanText(m.CameraName)
		}
		preview = append(preview, lbl)
	}
	suffix := ""
	if len(target) > 3 {
		suffix = fmt.Sprintf(", +%d lainnya", len(target)-3)
	}
	return fmt.Sprintf("%d CCTV (%s%s)", len(target), strings.Join(preview, ", "), suffix)
}

func formatTechnicianLine(tech *models.User) string {
	if tech == nil {
		return "Teknisi: Menunggu Penugasan Admin"
	}
	return fmt.Sprintf("Teknisi: %s (Otomatis)", cleanText(tech.FullName))
}

func formatWorkScopeLabel(scope string) string {
	switch scope {
	case WorkScopeSharedInfrastructure:
		return "Infrastruktur / NVR Bersama"
	case WorkScopeCameraFieldCheck:
		return "Cek Fisik Kamera"
	case WorkScopeUnitReplacement:
		return "Penggantian Unit"
	default:
		return "Penanganan Kamera"
	}
}

func formatTicketNewSingleMessage(ticketNo string, cam models.CCTV, origin string, eventAt time.Time, tech *models.User) string {
	header := fmt.Sprintf("TIKET BARU - %s", cleanText(ticketNo))
	timeLabel := "Mati sejak"
	if origin == OriginPostPlannedMaintenance {
		header += " (Pasca-Pemeliharaan Terjadwal)"
		timeLabel = "Akhir jadwal (SLA)"
	}
	return strings.Join([]string{
		header,
		"Kamera: " + formatSingleCameraLabel(cam),
		fmt.Sprintf("%s: %s", timeLabel, formatWITA(eventAt)),
		formatTechnicianLine(tech),
	}, "\n")
}

func formatTicketNewSharedNVRMessage(ticketNo, ipNVR string, camNames []string, confirmedAt time.Time, tech *models.User) string {
	preview := make([]string, 0, len(camNames))
	for idx, n := range camNames {
		if idx >= 5 {
			preview = append(preview, fmt.Sprintf("+%d lainnya", len(camNames)-5))
			break
		}
		preview = append(preview, cleanText(n))
	}
	return strings.Join([]string{
		fmt.Sprintf("GANGGUAN NVR - %s", cleanText(ticketNo)),
		fmt.Sprintf("NVR: %s (%s)", cleanText(ipNVR), formatWITA(confirmedAt)),
		fmt.Sprintf("Terdampak (%d): %s", len(camNames), strings.Join(preview, ", ")),
		formatTechnicianLine(tech),
	}, "\n")
}

func formatIncidentAssignedMessage(i Incident, camSummary, newTechName, prevTechName, workScope string) string {
	title := "PENUGASAN"
	techLine := fmt.Sprintf("Teknisi: %s (%s)", cleanText(newTechName), formatWorkScopeLabel(workScope))
	if strings.TrimSpace(prevTechName) != "" {
		title = "ALIH TUGAS"
		techLine = fmt.Sprintf("Dialihkan: %s -> %s", cleanText(prevTechName), cleanText(newTechName))
	}
	return strings.Join([]string{
		fmt.Sprintf("%s - %s", title, cleanText(i.TicketNumber)),
		"Kamera: " + camSummary,
		techLine,
	}, "\n")
}

func formatIncidentResolvedMessage(i Incident, camSummary, statusLabel string) string {
	title := "TIKET SELESAI"
	if i.ResolutionType != nil && *i.ResolutionType == ResolutionEscalatedReplacement {
		title = "TUTUP (GANTI UNIT)"
	}
	return strings.Join([]string{
		fmt.Sprintf("%s - %s", title, cleanText(i.TicketNumber)),
		"Kamera: " + camSummary,
		fmt.Sprintf("Status: %s", cleanText(statusLabel)),
	}, "\n")
}

func formatIncidentEscalatedClosedMessage(i Incident, camSummary string, escalatedNames []string) string {
	return strings.Join([]string{
		fmt.Sprintf("TUTUP (GANTI UNIT) - %s", cleanText(i.TicketNumber)),
		"Kamera: " + camSummary,
		fmt.Sprintf("Status: Nonaktif sementara menunggu ganti unit (%s)", cleanText(strings.Join(escalatedNames, ", "))),
	}, "\n")
}

func formatAdminReadyVerifyMessage(i Incident, camSummary string, recoveredCount, totalActive int) string {
	return strings.Join([]string{
		fmt.Sprintf("SIAP VERIFIKASI - %s", cleanText(i.TicketNumber)),
		"Kamera: " + camSummary,
		fmt.Sprintf("Status: %d/%d kamera stabil online 5 menit. Siap diverifikasi Admin.", recoveredCount, totalActive),
	}, "\n")
}

func formatAdminReminderMessage(i Incident, remLabel string, unrecovered int64) string {
	return strings.Join([]string{
		fmt.Sprintf("PENGINGAT ADMIN - %s", cleanText(i.TicketNumber)),
		fmt.Sprintf("Tiket: %s", cleanText(i.Title)),
		fmt.Sprintf("Status: Belum ada teknisi selama %s (%d kamera terganggu).", cleanText(remLabel), unrecovered),
	}, "\n")
}

func formatAdminEscalationMessage(i Incident, threshold time.Duration, unrecovered int64) string {
	return strings.Join([]string{
		fmt.Sprintf("ESKALASI DURASI - %s", cleanText(i.TicketNumber)),
		fmt.Sprintf("Tiket: %s", cleanText(i.Title)),
		fmt.Sprintf("Status: Belum pulih setelah %s (%d kamera masih terganggu).", cleanText(threshold.Round(time.Minute).String()), unrecovered),
	}, "\n")
}

func formatAdminMergedMessage(target Incident, mergedTickets []string, mergeReason string, totalActive int) string {
	return strings.Join([]string{
		fmt.Sprintf("TIKET DIGABUNG - %s", cleanText(target.TicketNumber)),
		fmt.Sprintf("Sumber: %s (total %d kamera aktif)", cleanText(strings.Join(mergedTickets, ", ")), totalActive),
		fmt.Sprintf("Alasan: %s", cleanText(mergeReason)),
	}, "\n")
}

func formatAdminCameraEscalatedMessage(i Incident, escalatedNames []string, remainingActive int) string {
	return strings.Join([]string{
		fmt.Sprintf("ESKALASI UNIT - %s", cleanText(i.TicketNumber)),
		fmt.Sprintf("Dinonaktifkan sementara: %s", cleanText(strings.Join(escalatedNames, ", "))),
		fmt.Sprintf("Sisa kamera dipantau pada tiket ini: %d", remainingActive),
	}, "\n")
}

func formatAdminMonitorDegradedMessage(cause, detail string, affectedCameras int) string {
	return strings.Join([]string{
		fmt.Sprintf("PEMANTAU TERGANGGU - %s", cleanText(cause)),
		fmt.Sprintf("Kamera unknown: %d unit (tiket offline palsu ditahan)", affectedCameras),
		fmt.Sprintf("Detail: %s", cleanText(detail)),
	}, "\n")
}

func formatAdminMonitorRecoveredMessage(cause string) string {
	return strings.Join([]string{
		fmt.Sprintf("PEMANTAU PULIH - %s", cleanText(cause)),
		"Status: Pemantau stabil 2 siklus berturut-turut; verifikasi normal dilanjutkan.",
	}, "\n")
}
