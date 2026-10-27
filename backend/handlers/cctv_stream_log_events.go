package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// cctvStreamLogDedupWindow adalah jendela deduplikasi event dengan tipe yang
// sama untuk satu CCTV. Probe pasif (2 menit) dan verifier aktif (15 menit)
// dapat sama-sama melihat transisi yang sama; log kedua dalam jendela ini
// dianggap duplikat.
const cctvStreamLogDedupWindow = 60 * time.Second

// shouldLogCCTVStreamEvent memutuskan apakah sebuah event layak ditulis ke
// cctv_stream_logs. Keputusan:
//   - standby tidak dicatat bila CCTV_LOG_STANDBY_EVENTS=false (default),
//     tetapi status stream tetap diperbarui oleh pemanggil.
//   - event dengan tipe sama seperti log terakhir untuk CCTV yang sama dalam
//     60 detik terakhir dianggap duplikat.
//   - kegagalan query dedup bersifat fail-open agar event penting tidak hilang.
func shouldLogCCTVStreamEvent(db *gorm.DB, cctvID uuid.UUID, eventType string, now time.Time) bool {
	if eventType == models.CCTVEventStandby && !config.CCTVLogStandbyEvents() {
		return false
	}
	if db == nil {
		return true
	}

	var last models.CCTVStreamLog
	err := db.Select("event_type", "created_at").
		Where("cctv_id = ?", cctvID).
		Order("created_at DESC").
		Limit(1).
		Find(&last).Error
	if err != nil {
		utils.Log.Warnf("⚠️ Gagal memeriksa log terakhir CCTV %s untuk deduplikasi: %v", cctvID, err)
		return true
	}

	if last.CreatedAt.IsZero() {
		return true
	}
	sameType := last.EventType == eventType
	withinWindow := now.Before(last.CreatedAt.Add(cctvStreamLogDedupWindow))
	return !sameType || !withinWindow
}
