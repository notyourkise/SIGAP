package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sigap-backend/models"
	"sigap-backend/utils"
)

const (
	// AmbiguousRetryFirstDelay adalah jeda retry pertama untuk pengiriman ambigu (konsisten 10 detik di Bagian 4 & 5).
	AmbiguousRetryFirstDelay  = 10 * time.Second
	AmbiguousRetrySecondDelay = 30 * time.Second
	MaxAmbiguousRetries       = 2
	MaxPendingAttempts        = 6
	MinGroupChatInterval      = 3200 * time.Millisecond // <= 18.75 pesan/menit per chat (< batas 20 pesan/menit Telegram)
	MaxTelegramMessageRunes   = 4000
)

var pendingBackoffSchedule = []time.Duration{
	5 * time.Second,
	15 * time.Second,
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
}

type DeliveryError struct {
	Ambiguous      bool
	Permanent      bool
	PreSendFailure bool
	RetryAfter     time.Duration
	Reason         string
}

func (e *DeliveryError) Error() string { return e.Reason }

type SendOptions struct {
	ParseMode        string
	ReplyToMessageID int64
}

type TelegramSender struct {
	Token   string
	Client  *http.Client
	baseURL string
}

// Send mengirim pesan ke Telegram Bot API dengan membedakan secara tegas:
//   - Kegagalan pra-kirim (DNS / dial TCP/TLS gagal sebelum request ditulis ke soket) -> PreSendFailure: true, Ambiguous: false.
//   - Kegagalan pasca-kirim (timeout baca soket, connection reset, JSON rusak, message_id/chat_id tidak cocok) -> Ambiguous: true.
//   - Penolakan permanen (400/401/403 atau konfigurasi salah) -> Permanent: true.
//
// Kredensial token bot tidak pernah disertakan pada pesan error.
func (s TelegramSender) Send(ctx context.Context, chatID, text string) (int64, error) {
	return s.SendWithOptions(ctx, chatID, text, SendOptions{})
}

func (s TelegramSender) SendWithOptions(ctx context.Context, chatID, text string, opts SendOptions) (int64, error) {
	destination, err := strconv.ParseInt(strings.TrimSpace(chatID), 10, 64)
	if err != nil || destination >= 0 || strings.TrimSpace(s.Token) == "" {
		return 0, &DeliveryError{Permanent: true, Reason: "Telegram: konfigurasi tujuan atau token tidak valid"}
	}
	payload := map[string]interface{}{
		"chat_id": strings.TrimSpace(chatID),
		"text":    text,
	}
	if strings.TrimSpace(opts.ParseMode) != "" {
		payload["parse_mode"] = strings.TrimSpace(opts.ParseMode)
	}
	if opts.ReplyToMessageID > 0 {
		payload["reply_to_message_id"] = opts.ReplyToMessageID
		payload["allow_sending_without_reply"] = true
	}
	body, _ := json.Marshal(payload)
	base := s.baseURL
	if base == "" {
		base = "https://api.telegram.org"
	}

	var wroteRequest bool
	var writeMu sync.Mutex
	traceCtx := httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				writeMu.Lock()
				wroteRequest = true
				writeMu.Unlock()
			}
		},
	})

	req, err := http.NewRequestWithContext(traceCtx, http.MethodPost, base+"/bot"+s.Token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return 0, &DeliveryError{Permanent: true, Reason: "Telegram: konfigurasi request tidak valid"}
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	response, err := client.Do(req)
	if err != nil {
		writeMu.Lock()
		sentBytes := wroteRequest
		writeMu.Unlock()
		if !sentBytes {
			return 0, &DeliveryError{
				Ambiguous:      false,
				PreSendFailure: true,
				Reason:         "Telegram: koneksi gagal sebelum pesan terkirim",
			}
		}
		return 0, &DeliveryError{
			Ambiguous: true,
			Reason:    "Telegram: hasil pengiriman belum dapat dipastikan",
		}
	}
	defer func() { _ = response.Body.Close() }()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
			Chat      struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"result"`
		Parameters struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return 0, &DeliveryError{Ambiguous: true, Reason: "Telegram: respons tidak dapat diverifikasi"}
	}
	if response.StatusCode != 200 || !result.OK {
		delay := time.Duration(result.Parameters.RetryAfter) * time.Second
		if delay < 0 {
			delay = 0
		}
		if delay > time.Hour {
			delay = time.Hour
		}
		return 0, &DeliveryError{
			Permanent:  response.StatusCode == 400 || response.StatusCode == 401 || response.StatusCode == 403,
			RetryAfter: delay,
			Reason:     fmt.Sprintf("Telegram: API menolak pengiriman (HTTP %d)", response.StatusCode),
		}
	}
	if result.Result.MessageID == 0 || result.Result.Chat.ID != destination {
		return 0, &DeliveryError{Ambiguous: true, Reason: "Telegram: identitas pesan atau grup tidak sesuai"}
	}
	return result.Result.MessageID, nil
}

// TelegramRouting memetakan 1 bot ke tujuan grup operasional teknisi per-site (ops_group)
// dan grup khusus Admin (admin_group) tanpa fallback lintas-site.
type TelegramRouting struct {
	SiteOpsChats   map[string]string
	SiteAdminChats map[string]string
	DefaultAdminID string
}

func (r TelegramRouting) ResolveDestination(siteID uuid.UUID, channel string) string {
	key := siteID.String()
	if channel == ChannelAdminGroup {
		if r.SiteAdminChats != nil {
			if c := strings.TrimSpace(r.SiteAdminChats[key]); c != "" {
				return c
			}
		}
		return strings.TrimSpace(r.DefaultAdminID)
	}
	if r.SiteOpsChats != nil {
		return strings.TrimSpace(r.SiteOpsChats[key])
	}
	return ""
}

func (r TelegramRouting) ConfiguredSites() (opsSites []string, hasAdmin bool) {
	for site, chat := range r.SiteOpsChats {
		if strings.TrimSpace(chat) != "" {
			opsSites = append(opsSites, site)
		}
	}
	if strings.TrimSpace(r.DefaultAdminID) != "" {
		hasAdmin = true
	}
	for _, chat := range r.SiteAdminChats {
		if strings.TrimSpace(chat) != "" {
			hasAdmin = true
		}
	}
	return opsSites, hasAdmin
}

// ChatRateLimiter menjaga jarak antar-pengiriman pada chat_id yang sama >= MinInterval (default 3,2 detik)
// agar patuh di bawah batas 20 pesan/menit per grup Telegram.
type ChatRateLimiter struct {
	mu          sync.Mutex
	MinInterval time.Duration
	lastSent    map[string]time.Time
}

func NewChatRateLimiter(minInterval time.Duration) *ChatRateLimiter {
	if minInterval <= 0 {
		minInterval = MinGroupChatInterval
	}
	return &ChatRateLimiter{
		MinInterval: minInterval,
		lastSent:    make(map[string]time.Time),
	}
}

func (l *ChatRateLimiter) AllowAt(chatID string, now time.Time) (allowed bool, nextAllowed time.Time) {
	if l == nil {
		return true, now
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lastSent == nil {
		l.lastSent = make(map[string]time.Time)
	}
	prev, ok := l.lastSent[chatID]
	if !ok || !now.Before(prev.Add(l.MinInterval)) {
		l.lastSent[chatID] = now
		return true, now
	}
	return false, prev.Add(l.MinInterval)
}

type claimedBatch struct {
	BatchID       uuid.UUID
	SiteID        uuid.UUID
	TargetChannel string
	Jobs          []Outbox
	WasAmbiguous  bool
}

func claimDueBatch(db *gorm.DB, now time.Time, routing TelegramRouting) (*claimedBatch, error) {
	opsSites, hasAdmin := routing.ConfiguredSites()
	if len(opsSites) == 0 && !hasAdmin {
		return nil, gorm.ErrRecordNotFound
	}

	var batch *claimedBatch
	err := db.Transaction(func(tx *gorm.DB) error {
		q := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		switch {
		case len(opsSites) > 0 && hasAdmin:
			q = q.Where("(target_channel = ? AND site_id IN ?) OR target_channel = ?", ChannelOpsGroup, opsSites, ChannelAdminGroup)
		case len(opsSites) > 0:
			q = q.Where("target_channel = ? AND site_id IN ?", ChannelOpsGroup, opsSites)
		default:
			q = q.Where("target_channel = ?", ChannelAdminGroup)
		}

		var head Outbox
		err := q.Where(
			"((status IN ('pending','ambiguous') AND next_attempt_at <= ? AND coalesce_until <= ?) OR (status = 'sending' AND lease_until < ?))",
			now, now, now,
		).Order("created_at ASC, id ASC").First(&head).Error
		if err != nil {
			return err
		}

		wasAmbiguous := head.Status == "ambiguous" || (head.Status == "sending" && head.LeaseUntil != nil && head.LeaseUntil.Before(now))
		lease := now.Add(30 * time.Second)
		batchID := uuid.New()
		jobs := []Outbox{head}

		// Penggabungan pesan (batching) untuk gelombang event sejenis pada site & channel yang sama.
		batchable := head.EventType == "CCTV_OFFLINE" || head.EventType == "INCIDENT_RESOLVED" || head.EventType == "CCTV_RECOVERED"
		if head.Status == "pending" && head.Attempts == 0 && batchable {
			var peers []Outbox
			if peerErr := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("id <> ? AND site_id = ? AND target_channel = ? AND event_type = ? AND status = 'pending' AND attempts = 0 AND next_attempt_at <= ?",
					head.ID, head.SiteID, head.TargetChannel, head.EventType, now).
				Order("created_at ASC, id ASC").
				Limit(24).
				Find(&peers).Error; peerErr == nil && len(peers) > 0 {
				totalRunes := len([]rune(head.Text)) + 160
				for _, p := range peers {
					addRunes := len([]rune(p.Text)) + 40
					if totalRunes+addRunes > MaxTelegramMessageRunes-250 {
						break
					}
					totalRunes += addRunes
					jobs = append(jobs, p)
				}
			}
		}

		for idx := range jobs {
			jobs[idx].LeaseUntil = &lease
			jobs[idx].Status = "sending"
			jobs[idx].Attempts++
			jobs[idx].LastAttemptAt = &now
			if len(jobs) > 1 {
				bCopy := batchID
				jobs[idx].BatchID = &bCopy
			}
			if err := tx.Save(&jobs[idx]).Error; err != nil {
				return err
			}
		}

		batch = &claimedBatch{
			BatchID:       batchID,
			SiteID:        head.SiteID,
			TargetChannel: head.TargetChannel,
			Jobs:          jobs,
			WasAmbiguous:  wasAmbiguous,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return batch, nil
}

func batchHeaderBadge(eventType string, count int) string {
	switch eventType {
	case "INCIDENT_RESOLVED":
		return fmt.Sprintf("[✅ %d TIKET SELESAI]", count)
	case "CCTV_RECOVERED":
		return fmt.Sprintf("[🟢 %d TIKET SIAP VERIFIKASI]", count)
	default:
		return fmt.Sprintf("[🚨 %d TIKET BARU]", count)
	}
}

func stripSingleItemBadgePrefix(text string) string {
	for _, prefix := range []string{
		"[🚨 TIKET BARU] ",
		"[🚨 GANGGUAN NVR] ",
		"[✅ TIKET SELESAI] ",
		"[🛠️ TUTUP • GANTI UNIT] ",
		"[🟢 SIAP VERIFIKASI] ",
		"[SIGAP] CCTV_OFFLINE\n",
	} {
		text = strings.TrimPrefix(text, prefix)
	}
	return text
}

func formatBatchText(b *claimedBatch) string {
	var text string
	if len(b.Jobs) == 1 {
		text = b.Jobs[0].Text
	} else {
		parts := make([]string, 0, len(b.Jobs)+1)
		parts = append(parts, fmt.Sprintf("<b>%s</b>", batchHeaderBadge(b.Jobs[0].EventType, len(b.Jobs))))
		for idx, j := range b.Jobs {
			cleaned := stripSingleItemBadgePrefix(j.Text)
			parts = append(parts, fmt.Sprintf("%d) %s", idx+1, cleaned))
		}
		text = strings.Join(parts, "\n\n")
	}

	head := b.Jobs[0]
	if b.WasAmbiguous || head.Attempts > 1 {
		ticketRef := "insiden terkait"
		if head.IncidentID != nil {
			ticketRef = head.IncidentID.String()[:8]
			if idx := strings.LastIndex(head.Text, "Tiket: "); idx != -1 {
				ticketRef = strings.TrimSpace(head.Text[idx+len("Tiket: "):])
			} else if idx := strings.Index(head.Text, "INC-"); idx != -1 {
				rest := head.Text[idx:]
				end := strings.IndexAny(rest, "< \n\t)")
				if end > 0 {
					ticketRef = rest[:end]
				} else if len(rest) > 0 {
					ticketRef = rest
				}
			}
		}
		if b.WasAmbiguous {
			text += fmt.Sprintf("\n<i>[Pengiriman ulang karena konfirmasi jaringan sebelumnya terputus — mengacu pada Tiket %s yang sama]</i>", escapeTG(ticketRef))
		} else {
			text += "\n<i>[Pengiriman ulang; mengacu pada tiket insiden yang sama]</i>"
		}
	}

	runes := []rune(text)
	if len(runes) > MaxTelegramMessageRunes {
		text = string(runes[:MaxTelegramMessageRunes])
	}
	return text
}

// DeliverOne mempertahankan kompatibilitas tanda tangan fungsi lama untuk peta site->ops_group.
func DeliverOne(ctx context.Context, db *gorm.DB, sender TelegramSender, chats map[string]string, now time.Time) (bool, error) {
	return DeliverWithRouting(ctx, db, sender, TelegramRouting{SiteOpsChats: chats}, nil, now)
}

// DeliverWithRouting mengklaim, menggabungkan (batching), membatasi laju per-chat, mengirim ke Telegram,
// serta memisahkan secara akurat status sent_at, failed_at, ambiguous retry (10s/30s), dan sla_breached.
func DeliverWithRouting(ctx context.Context, db *gorm.DB, sender TelegramSender, routing TelegramRouting, limiter *ChatRateLimiter, now time.Time) (bool, error) {
	batch, err := claimDueBatch(db.WithContext(ctx), now, routing)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	chat := routing.ResolveDestination(batch.SiteID, batch.TargetChannel)
	if chat == "" {
		for _, j := range batch.Jobs {
			_ = db.Model(&Outbox{}).Where("id = ?", j.ID).Updates(map[string]interface{}{
				"status":          "pending",
				"attempts":        gorm.Expr("GREATEST(attempts - 1, 0)"),
				"next_attempt_at": now.Add(time.Minute),
				"lease_until":     nil,
				"last_error":      "Tujuan grup Telegram untuk site/kanal ini belum dikonfigurasi",
			}).Error
		}
		return true, nil
	}

	if limiter != nil {
		if allowed, nextAllowed := limiter.AllowAt(chat, now); !allowed {
			restoreStatus := "pending"
			if batch.WasAmbiguous {
				restoreStatus = "ambiguous"
			}
			for _, j := range batch.Jobs {
				_ = db.Model(&Outbox{}).Where("id = ? AND status = 'sending'", j.ID).Updates(map[string]interface{}{
					"status":          restoreStatus,
					"attempts":        gorm.Expr("GREATEST(attempts - 1, 0)"),
					"next_attempt_at": nextAllowed,
					"lease_until":     nil,
				}).Error
			}
			return true, nil
		}
	}

	var replyToMsgID int64
	if len(batch.Jobs) == 1 && batch.Jobs[0].IncidentID != nil && batch.Jobs[0].EventType != "CCTV_OFFLINE" {
		_ = db.WithContext(ctx).Model(&Outbox{}).
			Select("message_id").
			Where("incident_id = ? AND target_channel = ? AND status = 'sent' AND message_id > 0",
				*batch.Jobs[0].IncidentID, batch.TargetChannel).
			Order("sent_at ASC, created_at ASC").
			Limit(1).
			Scan(&replyToMsgID).Error
	}

	text := formatBatchText(batch)
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	messageID, sendErr := sender.SendWithOptions(sendCtx, chat, text, SendOptions{
		ParseMode:        "HTML",
		ReplyToMessageID: replyToMsgID,
	})
	completedAt := now
	if realNow := time.Now().UTC(); realNow.After(completedAt) && completedAt.Sub(realNow) > -time.Minute {
		// Jika berjalan di waktu nyata (bukan jam simulasi masa depan/masa lalu jauh), pakai waktu konfirmasi aktual.
		completedAt = realNow
	}

	for _, job := range batch.Jobs {
		updates := map[string]interface{}{"lease_until": nil}
		if sendErr == nil {
			updates["status"] = "sent"
			updates["message_id"] = messageID
			updates["sent_at"] = completedAt
			updates["last_error"] = ""
			if job.SLADeadlineAt != nil && completedAt.After(*job.SLADeadlineAt) && !job.SLABreached {
				updates["sla_breached"] = true
				updates["sla_breached_at"] = completedAt
			}
		} else {
			status, nextTime, markFailed := classifyRetryOutcome(job, batch.WasAmbiguous, sendErr, now)
			updates["status"] = status
			updates["next_attempt_at"] = nextTime
			updates["last_error"] = sendErr.Error()
			if markFailed {
				updates["failed_at"] = now
			}
			if job.SLADeadlineAt != nil && now.After(*job.SLADeadlineAt) && !job.SLABreached {
				updates["sla_breached"] = true
				updates["sla_breached_at"] = now
			}
			if status == "failed" {
				ref := job.ID
				if job.IncidentID != nil {
					ref = *job.IncidentID
				}
				utils.NotifyRole(db, "admin", models.NotificationTypeIncidentDeliveryFailed,
					"Pengiriman Telegram Gagal Permanen",
					fmt.Sprintf("Pesan %s gagal dikirim ke Telegram (%s). Gunakan Kirim Ulang Manual setelah konfigurasi diperiksa.", job.EventType, sendErr.Error()),
					"incident", &ref)
			}
		}
		if uErr := db.Model(&Outbox{}).
			Where("id = ? AND status = 'sending' AND attempts = ?", job.ID, job.Attempts).
			Updates(updates).Error; uErr != nil {
			return true, uErr
		}
	}
	return true, nil
}

func classifyRetryOutcome(job Outbox, wasAmbiguous bool, sendErr error, now time.Time) (status string, nextAttempt time.Time, markFailed bool) {
	var delivery *DeliveryError
	isAmbiguous := wasAmbiguous
	isPermanent := false
	var retryAfter time.Duration

	if errors.As(sendErr, &delivery) {
		if delivery.Permanent {
			isPermanent = true
		}
		if delivery.Ambiguous {
			isAmbiguous = true
		}
		retryAfter = delivery.RetryAfter
	}

	if isPermanent {
		return "failed", now, true
	}

	if isAmbiguous {
		// Percobaan pertama yang menghasilkan ambigu adalah attempts=1 -> retry ke-1 berjeda 10s,
		// retry ke-2 berjeda 30s, dan setelah 2 retry ambigu otomatis berhenti (menunggu keputusan/retry manual Admin).
		// Catatan: retry pada status ambiguous tetap berpotensi menghasilkan pesan duplikat di grup Telegram.
		switch job.Attempts {
		case 1:
			delay := AmbiguousRetryFirstDelay
			if retryAfter > delay {
				delay = retryAfter
			}
			return "ambiguous", now.Add(delay), false
		case 2:
			delay := AmbiguousRetrySecondDelay
			if retryAfter > delay {
				delay = retryAfter
			}
			return "ambiguous", now.Add(delay), false
		default:
			// Hentikan retry otomatis dengan tetap mempertahankan status 'ambiguous' (next_attempt_at jauh di depan)
			// agar Admin dapat meninjau kemungkinan duplikasi atau menekan Kirim Ulang Manual.
			return "ambiguous", now.Add(365 * 24 * time.Hour), false
		}
	}

	if job.Attempts >= MaxPendingAttempts {
		return "failed", now, true
	}

	idx := job.Attempts - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(pendingBackoffSchedule) {
		idx = len(pendingBackoffSchedule) - 1
	}
	delay := pendingBackoffSchedule[idx]
	if retryAfter > delay {
		delay = retryAfter
	}
	return "pending", now.Add(delay), false
}

// ManualRetryOutbox mengizinkan Admin menjadwalkan pengiriman ulang manual untuk entri outbox
// yang berstatus 'failed' atau 'ambiguous' tanpa membuat tiket insiden baru.
func (s Service) ManualRetryOutbox(ctx context.Context, outboxID, site uuid.UUID, now time.Time) error {
	res := s.DB.WithContext(ctx).Model(&Outbox{}).
		Where("id = ? AND site_id = ? AND status IN ('failed', 'ambiguous')", outboxID, site).
		Updates(map[string]interface{}{
			"status":          gorm.Expr("CASE WHEN status = 'ambiguous' THEN 'ambiguous' ELSE 'pending' END"),
			"attempts":        gorm.Expr("CASE WHEN status = 'ambiguous' THEN 1 ELSE 0 END"),
			"next_attempt_at": now,
			"coalesce_until":  now,
			"lease_until":     gorm.Expr("NULL"),
			"failed_at":       gorm.Expr("NULL"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
