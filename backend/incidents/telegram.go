package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type DeliveryError struct {
	Ambiguous  bool
	Permanent  bool
	RetryAfter time.Duration
	Reason     string
}

func (e *DeliveryError) Error() string { return e.Reason }

type TelegramSender struct {
	Token   string
	Client  *http.Client
	baseURL string
}

func (s TelegramSender) Send(ctx context.Context, chatID, text string) (int64, error) {
	destination, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil || destination >= 0 || s.Token == "" {
		return 0, &DeliveryError{Permanent: true, Reason: "Telegram: konfigurasi tujuan atau token tidak valid"}
	}
	body, _ := json.Marshal(map[string]string{"chat_id": chatID, "text": text})
	base := s.baseURL
	if base == "" {
		base = "https://api.telegram.org"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/bot"+s.Token+"/sendMessage", bytes.NewReader(body))
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
		return 0, &DeliveryError{Ambiguous: true, Reason: "Telegram: hasil pengiriman belum dapat dipastikan"}
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
		return 0, &DeliveryError{Permanent: response.StatusCode == 400 || response.StatusCode == 401 || response.StatusCode == 403, RetryAfter: delay, Reason: "Telegram: API menolak pengiriman"}
	}
	if result.Result.MessageID == 0 || result.Result.Chat.ID != destination {
		return 0, &DeliveryError{Ambiguous: true, Reason: "Telegram: identitas pesan atau grup tidak sesuai"}
	}
	return result.Result.MessageID, nil
}

// DeliverOne uses a lease so a crashed worker does not lose queued messages.
// Site destinations are explicit: there is no cross-site fallback group.
func DeliverOne(ctx context.Context, db *gorm.DB, sender TelegramSender, chats map[string]string, now time.Time) (bool, error) {
	sites := make([]string, 0, len(chats))
	for site, chat := range chats {
		if chat != "" {
			sites = append(sites, site)
		}
	}
	if len(sites) == 0 {
		return false, nil
	}
	job, err := claim(db.WithContext(ctx), now, sites)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	chat := chats[job.SiteID.String()]
	if chat == "" {
		return true, db.Model(job).Updates(map[string]interface{}{"status": "pending", "attempts": gorm.Expr("attempts - 1"), "next_attempt_at": now.Add(time.Minute), "lease_until": nil, "last_error": "Grup Telegram untuk site ini belum dikonfigurasi"}).Error
	}
	text := job.Text
	if job.Attempts > 1 {
		text += "\n[Pengiriman ulang; pesan sebelumnya mungkin sudah diterima]"
	}
	if len([]rune(text)) > 4000 {
		text = string([]rune(text)[:4000])
	}
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	messageID, sendErr := sender.Send(sendCtx, chat, text)
	updates := map[string]interface{}{"lease_until": nil}
	if sendErr == nil {
		updates["status"] = "sent"
		updates["message_id"] = messageID
		updates["sent_at"] = time.Now().UTC()
		updates["last_error"] = ""
	} else {
		delay := time.Duration(1<<min(job.Attempts, 6)) * 5 * time.Second
		status := "pending"
		var delivery *DeliveryError
		if errors.As(sendErr, &delivery) {
			if delivery.Ambiguous {
				status = "ambiguous"
			}
			if delivery.RetryAfter > delay {
				delay = delivery.RetryAfter
			}
			if delivery.Permanent {
				status = "failed"
			}
		}
		if job.Attempts >= 6 {
			status = "failed"
		}
		updates["status"] = status
		updates["next_attempt_at"] = now.Add(delay)
		updates["last_error"] = sendErr.Error()
	}
	return true, db.Model(&Outbox{}).Where("id = ? AND status = 'sending' AND attempts = ?", job.ID, job.Attempts).Updates(updates).Error
}
