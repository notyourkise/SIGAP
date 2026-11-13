package incidents

import (
	"fmt"
	"html"
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

func escapeTG(s string) string {
	return html.EscapeString(strings.TrimSpace(s))
}

func formatSingleCameraLabel(cam models.CCTV) string {
	code := strings.TrimSpace(cam.KodeCCTV)
	name := strings.TrimSpace(cam.Nama)
	loc := strings.TrimSpace(cam.LokasiArea)

	base := escapeTG(name)
	if code != "" && name != "" {
		base = fmt.Sprintf("%s — %s", escapeTG(code), escapeTG(name))
	} else if code != "" {
		base = escapeTG(code)
	}
	if loc != "" {
		return fmt.Sprintf("<b>%s</b> (%s)", base, escapeTG(loc))
	}
	return fmt.Sprintf("<b>%s</b>", base)
}

func formatMembersSummary(members []Member, fallbackTitle string) string {
	if len(members) == 0 {
		return fmt.Sprintf("<b>%s</b>", escapeTG(fallbackTitle))
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
		base := escapeTG(m.CameraName)
		if m.CameraCode != "" && m.CameraName != "" {
			base = fmt.Sprintf("%s — %s", escapeTG(m.CameraCode), escapeTG(m.CameraName))
		} else if m.CameraCode != "" {
			base = escapeTG(m.CameraCode)
		}
		if strings.TrimSpace(m.LocationArea) != "" {
			return fmt.Sprintf("<b>%s</b> (%s)", base, escapeTG(m.LocationArea))
		}
		return fmt.Sprintf("<b>%s</b>", base)
	}

	preview := make([]string, 0, 3)
	for idx, m := range target {
		if idx >= 3 {
			break
		}
		lbl := m.CameraCode
		if lbl == "" {
			lbl = m.CameraName
		}
		preview = append(preview, escapeTG(lbl))
	}
	suffix := ""
	if len(target) > 3 {
		suffix = fmt.Sprintf(", +%d lainnya", len(target)-3)
	}
	return fmt.Sprintf("<b>%d CCTV</b> (%s%s)", len(target), strings.Join(preview, ", "), suffix)
}

func formatTechnicianLine(tech *models.User) string {
	if tech == nil {
		return "⏳ Teknisi: Menunggu Penugasan Admin"
	}
	return fmt.Sprintf("👷 Teknisi: <b>%s</b> (Otomatis)", escapeTG(tech.FullName))
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
	header := fmt.Sprintf("[🚨 TIKET BARU] <b>%s</b>", escapeTG(ticketNo))
	timeLabel := "Mati sejak"
	if origin == OriginPostPlannedMaintenance {
		header += " • [Pasca-Pemeliharaan Terjadwal]"
		timeLabel = "Akhir jadwal (SLA)"
	}
	return strings.Join([]string{
		header,
		"📹 " + formatSingleCameraLabel(cam),
		fmt.Sprintf("🕒 %s: %s", timeLabel, formatWITA(eventAt)),
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
		preview = append(preview, escapeTG(n))
	}
	return strings.Join([]string{
		fmt.Sprintf("[🚨 GANGGUAN NVR] <b>%s</b>", escapeTG(ticketNo)),
		fmt.Sprintf("🖥️ NVR <b>%s</b> tidak merespons • %s", escapeTG(ipNVR), formatWITA(confirmedAt)),
		fmt.Sprintf("📹 Terdampak (%d): %s", len(camNames), strings.Join(preview, ", ")),
		formatTechnicianLine(tech),
	}, "\n")
}

func formatIncidentAssignedMessage(i Incident, camSummary, newTechName, prevTechName, workScope string) string {
	badge := "[👷 PENUGASAN]"
	techLine := fmt.Sprintf("👷 Teknisi: <b>%s</b> (%s)", escapeTG(newTechName), escapeTG(formatWorkScopeLabel(workScope)))
	if strings.TrimSpace(prevTechName) != "" {
		badge = "[🔄 ALIH TUGAS]"
		techLine = fmt.Sprintf("👷 Dialihkan: %s → <b>%s</b>", escapeTG(prevTechName), escapeTG(newTechName))
	}
	return strings.Join([]string{
		fmt.Sprintf("%s <b>%s</b>", badge, escapeTG(i.TicketNumber)),
		"📹 " + camSummary,
		techLine,
	}, "\n")
}

func formatIncidentResolvedMessage(i Incident, camSummary, statusLabel string) string {
	badge := "[✅ TIKET SELESAI]"
	if i.ResolutionType != nil && *i.ResolutionType == ResolutionEscalatedReplacement {
		badge = "[🛠️ TUTUP • GANTI UNIT]"
	}
	return strings.Join([]string{
		fmt.Sprintf("%s <b>%s</b>", badge, escapeTG(i.TicketNumber)),
		"📹 " + camSummary,
		fmt.Sprintf("📌 Status: %s", escapeTG(statusLabel)),
	}, "\n")
}

func formatIncidentEscalatedClosedMessage(i Incident, camSummary string, escalatedNames []string) string {
	return strings.Join([]string{
		fmt.Sprintf("[🛠️ TUTUP • GANTI UNIT] <b>%s</b>", escapeTG(i.TicketNumber)),
		"📹 " + camSummary,
		fmt.Sprintf("📌 Nonaktif sementara menunggu ganti unit: %s", escapeTG(strings.Join(escalatedNames, ", "))),
	}, "\n")
}

func formatAdminReadyVerifyMessage(i Incident, camSummary string, recoveredCount, totalActive int) string {
	return strings.Join([]string{
		fmt.Sprintf("[🟢 SIAP VERIFIKASI] <b>%s</b>", escapeTG(i.TicketNumber)),
		"📹 " + camSummary,
		fmt.Sprintf("📌 %d/%d kamera stabil online 5 menit. Siap diverifikasi Admin.", recoveredCount, totalActive),
	}, "\n")
}

func formatAdminReminderMessage(i Incident, remLabel string, unrecovered int64) string {
	return strings.Join([]string{
		fmt.Sprintf("[🔔 PENGINGAT ADMIN] <b>%s</b>", escapeTG(i.TicketNumber)),
		fmt.Sprintf("📹 %s", escapeTG(i.Title)),
		fmt.Sprintf("⏳ Belum ada teknisi selama %s (%d kamera terganggu).", escapeTG(remLabel), unrecovered),
	}, "\n")
}

func formatAdminEscalationMessage(i Incident, threshold time.Duration, unrecovered int64) string {
	return strings.Join([]string{
		fmt.Sprintf("[⏰ ESKALASI DURASI] <b>%s</b>", escapeTG(i.TicketNumber)),
		fmt.Sprintf("📹 %s", escapeTG(i.Title)),
		fmt.Sprintf("⏳ Belum pulih setelah %s (%d kamera masih terganggu).", escapeTG(threshold.Round(time.Minute).String()), unrecovered),
	}, "\n")
}

func formatAdminMergedMessage(target Incident, mergedTickets []string, mergeReason string, totalActive int) string {
	return strings.Join([]string{
		fmt.Sprintf("[🔗 TIKET DIGABUNG] <b>%s</b>", escapeTG(target.TicketNumber)),
		fmt.Sprintf("📎 Sumber: %s (total %d kamera aktif)", escapeTG(strings.Join(mergedTickets, ", ")), totalActive),
		fmt.Sprintf("📝 Alasan: %s", escapeTG(mergeReason)),
	}, "\n")
}

func formatAdminCameraEscalatedMessage(i Incident, escalatedNames []string, remainingActive int) string {
	return strings.Join([]string{
		fmt.Sprintf("[🛠️ ESKALASI UNIT] <b>%s</b>", escapeTG(i.TicketNumber)),
		fmt.Sprintf("📹 Dinonaktifkan sementara: %s", escapeTG(strings.Join(escalatedNames, ", "))),
		fmt.Sprintf("📌 Sisa kamera dipantau pada tiket ini: %d", remainingActive),
	}, "\n")
}

func formatAdminMonitorDegradedMessage(cause, detail string, affectedCameras int) string {
	return strings.Join([]string{
		fmt.Sprintf("[⚠️ PEMANTAU TERGANGGU] <b>%s</b>", escapeTG(cause)),
		fmt.Sprintf("📹 Kamera unknown: %d unit (tiket offline palsu ditahan)", affectedCameras),
		fmt.Sprintf("📝 Detail: %s", escapeTG(detail)),
	}, "\n")
}

func formatAdminMonitorRecoveredMessage(cause string) string {
	return strings.Join([]string{
		fmt.Sprintf("[💚 PEMANTAU PULIH] <b>%s</b>", escapeTG(cause)),
		"📌 Pemantau stabil 2 siklus berturut-turut; verifikasi normal dilanjutkan.",
	}, "\n")
}
