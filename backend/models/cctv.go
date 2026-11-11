package models

import (
	"time"

	"github.com/google/uuid"
)

// Nilai StreamState. Status idle berarti path on-demand terdaftar tetapi
// sumber belum berjalan karena tidak ada penonton; ini bukan bukti offline.
const (
	StreamStateLive    = "live"
	StreamStateIdle    = "idle"
	StreamStateOffline = "offline"
	StreamStateUnknown = "unknown"
)

// CCTV mewakili master data perangkat CCTV tanpa menyimpan kredensial
type CCTV struct {
	Base
	KodeCCTV      string   `gorm:"type:varchar(50);uniqueIndex;not null" json:"kode_cctv"`
	IDChannel     string   `gorm:"type:varchar(50)" json:"id_channel"`
	Nama          string   `gorm:"type:varchar(100);not null" json:"nama"`
	LokasiArea    string   `gorm:"type:varchar(255)" json:"lokasi_area"`
	Longitude     *float64 `gorm:"type:double precision" json:"longitude"`
	Latitude      *float64 `gorm:"type:double precision" json:"latitude"`
	JenisCCTV     string   `gorm:"type:varchar(50)" json:"jenis_cctv"`
	SumberListrik string   `gorm:"type:varchar(50)" json:"sumber_listrik"`
	TipeKoneksi   string   `gorm:"type:varchar(50)" json:"tipe_koneksi"`
	IndoorOutdoor string   `gorm:"type:varchar(20)" json:"indoor_outdoor"`
	Pemilik       string   `gorm:"type:varchar(100)" json:"pemilik"`
	Status        string   `gorm:"type:varchar(20);not null" json:"status"`
	IPNVR         string   `gorm:"type:varchar(45)" json:"ip_nvr"`
	IPCamera      string   `gorm:"type:varchar(45)" json:"ip_camera"`
	Keterangan    string   `gorm:"type:text" json:"keterangan"`
	QrPayload     string   `gorm:"type:text;unique" json:"qr_payload"`
	// HealthScore menyimpan skor kesehatan terbaru dari kalkulator operasional.
	// Nilai awal 100 dan dibatasi pada rentang 0 sampai 100 oleh kalkulator.
	HealthScore int `gorm:"type:integer;not null;default:100" json:"health_score"`
	// === Field Streaming RTSP (Task 9.1) ===
	// ChannelPort adalah nomor channel kamera pada NVR (mis. 19 untuk CH 19).
	ChannelPort       int    `gorm:"type:integer" json:"channel_port"`
	RTSPPassword      string `gorm:"type:text" json:"rtsp_password,omitempty"`
	StreamEnabled     bool   `gorm:"type:boolean;not null;default:false" json:"stream_enabled"`
	MonitoringEnabled bool   `gorm:"column:monitoring_enabled;type:boolean;not null;default:true" json:"monitoring_enabled"`
	// PlannedOutage* adalah metadata computed saat CCTV sedang berada dalam
	// jendela pemeliharaan terjadwal yang dibuat Admin ("Sedang pemeliharaan terjadwal").
	PlannedOutageActive bool       `gorm:"-" json:"planned_outage_active,omitempty"`
	PlannedOutageID     *uuid.UUID `gorm:"-" json:"planned_outage_id,omitempty"`
	PlannedOutageReason string     `gorm:"-" json:"planned_outage_reason,omitempty"`
	PlannedOutageEndsAt *time.Time `gorm:"-" json:"planned_outage_ends_at,omitempty"`
	// RTSPURL adalah URL RTSP penuh per-CCTV, mis.
	//   rtsp://USER:PASSWORD@NVR_HOST:554/Streaming/channels/1901
	// Bila diisi, URL ini dipakai langsung oleh MediaMTX sebagai sumber stream,
	// menggantikan penyusunan otomatis dari ChannelPort + kredensial NVR global.
	// Di database nilai ini disimpan terenkripsi (prefix enc:v1:); data legacy
	// plaintext tetap dapat dibaca. Jangan pernah mengembalikan nilai mentah ke
	// klien — gunakan RTSPURLMasked/RTSPURLConfigured pada response.
	RTSPURL string `gorm:"type:text" json:"rtsp_url,omitempty"`
	// RTSPURLMasked adalah URL tersamarkan (password diganti xxxxx) untuk UI.
	// Field computed, tidak disimpan di database.
	RTSPURLMasked string `gorm:"-" json:"rtsp_url_masked,omitempty"`
	// RTSPURLConfigured menandai apakah CCTV sudah memiliki URL RTSP tersimpan.
	// Frontend memakai flag ini agar tidak perlu menerima nilai rahasia saat edit.
	RTSPURLConfigured bool `gorm:"-" json:"rtsp_url_configured,omitempty"`
	// SnapshotURL adalah path URL snapshot terbaru (computed, tidak disimpan di DB).
	// Diisi saat handler mengembalikan data CCTV agar frontend dapat langsung
	// mengambil snapshot terbaru melalui endpoint /cctv/{id}/snapshot.
	SnapshotURL string `gorm:"-" json:"snapshot_url,omitempty"`
	// ZoneID menunjuk ke zona operasional tempat CCTV berada (Task 11.4,
	// Sprint 28). Opsional — CCTV dapat belum dimasukkan ke zona mana pun.
	ZoneID *uuid.UUID `gorm:"type:uuid;index" json:"zone_id,omitempty"`
	// Zone adalah relasi one-to-many ke master zona (preload untuk detail).
	Zone *Zone `gorm:"foreignKey:ZoneID" json:"zone,omitempty"`
	// SiteID menunjuk ke kawasan operasional (multi-site, Task 12.2). Opsional
	// — CCTV dapat belum dimasukkan ke site mana pun; data existing otomatis
	// masuk ke site default "OIKN" saat migrasi.
	SiteID *uuid.UUID `gorm:"type:uuid;index" json:"site_id,omitempty"`
	// Site dimuat pada endpoint daftar/detail agar antarmuka dapat menampilkan
	// nama kawasan, bukan UUID mentah.
	Site *Site `gorm:"foreignKey:SiteID" json:"site,omitempty"`
	// === Field Status Live Stream (Task 14A.1, Sprint 37) ===
	// StreamLive mencerminkan hasil probe RTSP terakhir: true bila kamera
	// merespons probe (dianggap live), false bila down/belum pernah diperiksa.
	// Dipertahankan untuk kompatibilitas; gunakan StreamState untuk status
	// yang lebih tepat (idle/offline/unknown).
	StreamLive bool `gorm:"column:stream_live;default:false" json:"stream_live"`
	// StreamState membedakan status stream: live, idle (path on-demand terdaftar
	// tetapi tidak ada penonton/sumber belum siap), offline (kegagalan terkonfirmasi),
	// atau unknown (belum dapat disimpulkan). Nilai default unknown.
	StreamState string `gorm:"column:stream_state;type:varchar(20);not null;default:'unknown'" json:"stream_state"`
	// StreamCheckedAt mencatat waktu pemeriksaan (probe) terakhir. Nilai nil
	// berarti CCTV belum pernah diperiksa oleh probe engine.
	StreamCheckedAt *time.Time `gorm:"column:stream_checked_at" json:"stream_checked_at,omitempty"`
	// PingLatencyMs menyimpan latensi ping/probe terakhir (ms) — HANYA untuk
	// kamera Metro-E (jaringan dedicated OIKN). Nilai nil bila belum pernah
	// diprobe atau kamera GSM (tidak diprobe).
	PingLatencyMs *int `gorm:"column:ping_latency_ms" json:"ping_latency_ms,omitempty"`
	// StreamLastActiveAt mencatat waktu terakhir stream kamera aktif
	// (Ready di MediaMTX). Dipakai badge GSM ("Pernah aktif hari ini").
	StreamLastActiveAt *time.Time `gorm:"column:stream_last_active_at" json:"stream_last_active_at,omitempty"`
	// === Field Verifikasi Aktif Stream (Sprint 41) ===
	// StreamVerifiedState adalah hasil verifikasi end-to-end melalui MediaMTX,
	// terpisah dari StreamState yang bersifat pasif. Nilai: live (sumber benar-
	// benar mengirim track), offline (gagal setelah beberapa percobaan), atau
	// unknown (belum diverifikasi / MediaMTX tidak dapat dihubungi).
	StreamVerifiedState string `gorm:"column:stream_verified_state;type:varchar(20);not null;default:'unknown'" json:"stream_verified_state"`
	// StreamVerifiedAt mencatat waktu percobaan verifikasi terakhir. Nilai nil
	// berarti kamera belum pernah diverifikasi secara aktif.
	StreamVerifiedAt *time.Time `gorm:"column:stream_verified_at" json:"stream_verified_at,omitempty"`
	// StreamVerifiedMethod menyimpan metode verifikasi terakhir, mis.
	// "mediamtx-rtsp", "mediamtx-live", atau "mediamtx-hls".
	StreamVerifiedMethod string `gorm:"column:stream_verified_method;type:varchar(32);not null;default:''" json:"stream_verified_method,omitempty"`
	// StreamVerifyFailures menghitung kegagalan verifikasi berturut-turut.
	// Offline baru ditetapkan setelah ambang tertentu agar tidak false alarm.
	StreamVerifyFailures int `gorm:"column:stream_verify_failures;not null;default:0" json:"stream_verify_failures"`
	// StreamVerifyError menyimpan ringkasan kegagalan terakhir (tanpa kredensial)
	// untuk membantu diagnosis di UI/API.
	StreamVerifyError string `gorm:"column:stream_verify_error;type:text;not null;default:''" json:"stream_verify_error,omitempty"`
}
