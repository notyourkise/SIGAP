package models

import "github.com/google/uuid"

// Nama event webhook yang didukung sistem (Task 12.3, Sprint 29).
const (
	// EventWebhookCCTVStatusChanged dipicu saat status CCTV berubah.
	EventWebhookCCTVStatusChanged = "cctv.status_changed"
	// EventWebhookMaintenanceCompleted dipicu saat tugas pemeliharaan selesai.
	EventWebhookMaintenanceCompleted = "maintenance.completed"
	// EventWebhookMaintenanceOverdue dipicu saat tugas pemeliharaan terlambat.
	EventWebhookMaintenanceOverdue = "maintenance.overdue"
	// EventWebhookHealthScoreChanged dipicu saat skor kesehatan CCTV berubah.
	EventWebhookHealthScoreChanged = "health_score.changed"
	// EventWebhookStreamDown dipicu saat stream CCTV down.
	EventWebhookStreamDown = "stream.down"
)

// WebhookEndpoint adalah target URL penerima event webhook (Task 12.3).
// Setiap endpoint berlangganan daftar event dan memakai Secret untuk HMAC.
type WebhookEndpoint struct {
	Base
	// Name adalah label pengenal endpoint.
	Name string `gorm:"type:varchar(100);not null" json:"name"`
	// URL adalah target endpoint HTTPS yang menerima POST payload event.
	URL string `gorm:"type:text;not null" json:"url"`
	// Events adalah daftar event yang dilanggan (jsonb array).
	Events []string `gorm:"type:jsonb;serializer:json;not null;default:'[]'" json:"events"`
	// Secret dipakai sebagai kunci HMAC-SHA256 untuk header X-Signature-256.
	// Disimpan sebagai plaintext agar signature dapat dihitung ulang.
	Secret string `gorm:"type:text;not null" json:"-"`
	// IsActive menandakan endpoint aktif menerima pengiriman.
	IsActive bool `gorm:"type:boolean;not null;default:true" json:"is_active"`
	// MaxLogRetentionDays adalah berapa hari log delivery disimpan sebelum
	// dihapus oleh cleanup periodik (default 30). (Task 13.10, Sprint 34)
	MaxLogRetentionDays int `gorm:"type:integer;not null;default:30" json:"max_log_retention_days"`
	// CreatedBy menunjuk user admin pembuat endpoint.
	CreatedBy uuid.UUID `gorm:"type:uuid;not null;index" json:"created_by"`
	User      User      `gorm:"foreignKey:CreatedBy" json:"user,omitempty"`
}

// WebhookDelivery mencatat satu percobaan pengiriman event ke endpoint.
// Status: pending / success / failed. Menyimpan detail respons & latensi.
type WebhookDelivery struct {
	Base
	// WebhookEndpointID menunjuk endpoint tujuan.
	WebhookEndpointID uuid.UUID       `gorm:"type:uuid;not null;index" json:"webhook_endpoint_id"`
	WebhookEndpoint   WebhookEndpoint `gorm:"foreignKey:WebhookEndpointID" json:"webhook_endpoint,omitempty"`
	// Event adalah nama event yang dikirim.
	Event string `gorm:"type:varchar(100);not null;index" json:"event"`
	// Payload adalah body JSON yang dikirim (string).
	Payload string `gorm:"type:jsonb;not null" json:"payload"`
	// ResponseCode adalah HTTP status code dari endpoint penerima (0 bila gagal).
	ResponseCode int `gorm:"type:integer" json:"response_code"`
	// ResponseBody adalah ringkasan body respons penerima.
	ResponseBody string `gorm:"type:text" json:"response_body"`
	// Latency adalah durasi pengiriman dalam milidetik.
	Latency int64 `gorm:"type:bigint" json:"latency_ms"`
	// Status: pending / success / failed.
	Status string `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	// Attempt mencatat jumlah percobaan yang sudah dilakukan (maks 3).
	Attempt int `gorm:"type:integer;not null;default:1" json:"attempt"`
}

// WebhookDeliverySuccess menandakan pengiriman berhasil (status 2xx).
const WebhookDeliverySuccess = "success"

// WebhookDeliveryFailed menandakan pengiriman gagal setelah retry habis.
const WebhookDeliveryFailed = "failed"

// WebhookDeliveryPending menandakan pengiriman belum selesai / sedang diproses.
const WebhookDeliveryPending = "pending"

func (WebhookDelivery) TableName() string {
	return "webhook_deliveries"
}
