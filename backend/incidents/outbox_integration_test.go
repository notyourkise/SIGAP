package incidents

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"sigap-backend/models"
)

func TestPostgresOutboxPersistsTelegramReceiptAndCoalesceBatching(t *testing.T) {
	db, camera := integrationDB(t)
	now := time.Now().UTC()
	svc := Service{DB: db}
	ctx := context.Background()

	// Buka 25 tiket offline pada site yang sama dalam jendela penggabungan 8 detik.
	siteID := *camera.SiteID
	if err := svc.Observe(ctx, camera, StateOffline, now); err != nil {
		t.Fatal(err)
	}
	for idx := 2; idx <= 25; idx++ {
		cid := uuid.New()
		cam := models.CCTV{
			Base:              models.Base{ID: cid},
			KodeCCTV:          fmt.Sprintf("BATCH-%02d-%s", idx, cid.String()[:6]),
			QrPayload:         cid.String(),
			Nama:              fmt.Sprintf("Kamera Batch %02d", idx),
			SiteID:            &siteID,
			Status:            "aktif",
			MonitoringEnabled: true,
		}
		if err := db.Create(&cam).Error; err != nil {
			t.Fatal(err)
		}
		if err := svc.Observe(ctx, cam, StateOffline, now.Add(time.Duration(idx)*100*time.Millisecond)); err != nil {
			t.Fatal(err)
		}
	}

	var mu sync.Mutex
	var receivedBodies []string
	var msgSeq int64
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		msgSeq++
		seq := msgSeq
		receivedBodies = append(receivedBodies, string(raw))
		mu.Unlock()
		_, _ = fmt.Fprintf(w, `{"ok":true,"result":{"message_id":%d,"chat":{"id":-100}}}`, seq)
	}))
	defer receiver.Close()

	sender := TelegramSender{Token: "test", Client: receiver.Client(), baseURL: receiver.URL}
	routing := TelegramRouting{SiteOpsChats: map[string]string{siteID.String(): "-100"}}
	limiter := NewChatRateLimiter(MinGroupChatInterval)

	// Sebelum coalesce_until (8s) lewat, belum boleh ada pengiriman keluar.
	workedEarly, err := DeliverWithRouting(ctx, db, sender, routing, limiter, now.Add(2*time.Second))
	if err != nil || workedEarly {
		t.Fatalf("expected hold during 8s coalesce window, got worked=%v err=%v", workedEarly, err)
	}

	// Setelah coalesce_until lewat, 25 tiket digabung menjadi <= 2 pesan HTTP (jauh di bawah batas 20 pesan/menit!).
	simTime := now.Add(12 * time.Second)
	for step := 0; step < 10; step++ {
		worked, err := DeliverWithRouting(ctx, db, sender, routing, limiter, simTime)
		if err != nil {
			t.Fatal(err)
		}
		if !worked {
			break
		}
		simTime = simTime.Add(4 * time.Second)
	}

	var sentCount int64
	db.Model(&Outbox{}).Where("site_id = ? AND status = 'sent'", siteID).Count(&sentCount)
	if sentCount != 25 {
		t.Fatalf("expected all 25 outbox rows marked sent, got %d", sentCount)
	}
	if len(receivedBodies) == 0 || len(receivedBodies) > 3 {
		t.Fatalf("expected 25 tickets batched into 1..3 Telegram messages (<20/min), got %d", len(receivedBodies))
	}
}

func TestPostgresAmbiguousRetrySLABreachAndSingleReminders(t *testing.T) {
	db, camera := integrationDB(t)
	ctx := context.Background()
	siteID := *camera.SiteID
	zeroCoalesce := time.Duration(0)
	svc := Service{DB: db, CoalesceWindow: &zeroCoalesce}
	t0 := time.Now().UTC().Truncate(time.Microsecond).Add(-5 * time.Hour)

	if err := svc.Observe(ctx, camera, StateOffline, t0); err != nil {
		t.Fatal(err)
	}

	// 1. Server yang memutus koneksi pasca-request -> status 'ambiguous', next_attempt_at +10s, lalu +30s,
	//    dan saat akhirnya diterima harus memuat peringatan eksplisit potensi duplikasi.
	var attempt int
	var mu sync.Mutex
	var lastText string
	dropThenSucceedLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dropThenSucceedLn.Close() }()

	go func() {
		for {
			conn, acceptErr := dropThenSucceedLn.Accept()
			if acceptErr != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				req, rErr := http.ReadRequest(bufio.NewReader(c))
				if rErr != nil {
					return
				}
				bodyBytes, _ := io.ReadAll(req.Body)
				mu.Lock()
				attempt++
				cur := attempt
				lastText = string(bodyBytes)
				mu.Unlock()
				if cur == 1 {
					// Putus koneksi pasca-request (mensimulasikan timeout/disconnect ambigu).
					return
				}
				resp := `{"ok":true,"result":{"message_id":777,"chat":{"id":-100}}}`
				_, _ = fmt.Fprintf(c, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(resp), resp)
			}(conn)
		}
	}()

	sender := TelegramSender{
		Token:   "test",
		Client:  &http.Client{Timeout: 2 * time.Second},
		baseURL: "http://" + dropThenSucceedLn.Addr().String(),
	}
	routing := TelegramRouting{
		SiteOpsChats:   map[string]string{siteID.String(): "-100"},
		DefaultAdminID: "-200",
	}

	// Percobaan 1 pada t0+1s -> gagal pasca-kirim -> status='ambiguous', next_attempt_at = t0+11s (+10s).
	if _, err := DeliverWithRouting(ctx, db, sender, routing, nil, t0.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var job Outbox
	db.Where("site_id = ?", siteID).First(&job)
	if job.Status != "ambiguous" || job.SentAt != nil || job.NextAttemptAt.Sub(t0.Add(time.Second)) != 10*time.Second {
		t.Fatalf("expected ambiguous with 10s retry and nil sent_at, got %+v", job)
	}

	// Sebelum 10s (pada t0+6s), tidak boleh dikirim ulang.
	if worked, _ := DeliverWithRouting(ctx, db, sender, routing, nil, t0.Add(6*time.Second)); worked {
		t.Fatal("ambiguous retry fired before 10s delay")
	}

	// Pada t0+11s -> berhasil dikirim dengan penanda eksplisit retry ambigu!
	if _, err := DeliverWithRouting(ctx, db, sender, routing, nil, t0.Add(11*time.Second)); err != nil {
		t.Fatal(err)
	}
	db.First(&job, "id = ?", job.ID)
	if job.Status != "sent" || job.SentAt == nil || job.SLABreached {
		t.Fatalf("expected sent within 5m SLA without breach, got %+v", job)
	}
	mu.Lock()
	deliveredPayload := lastText
	mu.Unlock()
	if !strings.Contains(deliveredPayload, "konfirmasi jaringan sebelumnya terputus") {
		t.Fatalf("ambiguous retry must include duplicate warning text, got: %s", deliveredPayload)
	}

	// 2. Uji pelanggaran SLA (Koreksi #3): tiket kedua yang gagal permanen (401) dan melewati tenggat 5 menit
	//    wajib ditandai sla_breached = true meskipun tidak pernah berhasil terkirim, serta ManualRetryOutbox berfungsi.
	cid2 := uuid.New()
	cam2 := models.CCTV{
		Base:              models.Base{ID: cid2},
		KodeCCTV:          "SLA-FAIL-" + cid2.String()[:6],
		QrPayload:         cid2.String(),
		Nama:              "Kamera SLA Gagal",
		SiteID:            &siteID,
		Status:            "aktif",
		MonitoringEnabled: true,
	}
	if err := db.Create(&cam2).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Observe(ctx, cam2, StateOffline, t0); err != nil {
		t.Fatal(err)
	}
	unauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"ok":false,"description":"unauthorized"}`)
	}))
	defer unauthSrv.Close()
	badSender := TelegramSender{Token: "bad", Client: unauthSrv.Client(), baseURL: unauthSrv.URL}
	if _, err := DeliverWithRouting(ctx, db, badSender, routing, nil, t0.Add(20*time.Second)); err != nil {
		t.Fatal(err)
	}

	var failedJob Outbox
	db.Joins("JOIN cctv_incidents i ON i.id = telegram_outbox.incident_id").
		Joins("JOIN cctv_incident_cameras m ON m.incident_id = i.id").
		Where("m.cctv_id = ?", cam2.ID).First(&failedJob)
	if failedJob.Status != "failed" || failedJob.FailedAt == nil || failedJob.SentAt != nil {
		t.Fatalf("expected failed status with failed_at and nil sent_at, got %+v", failedJob)
	}

	// Evaluasi pada t0 + 4 jam + 10 menit:
	//   - failedJob melewati tenggat 5 menit tanpa sent_at -> sla_breached = true!
	//   - Pengingat 30 menit & eskalasi 4 jam masing-masing diantrekan MAKSIMAL 1 kali per insiden walau dievaluasi berulang!
	evalAt := t0.Add(4*time.Hour + 10*time.Minute)
	breaches1, rem1, esc1, err := svc.EvaluateRemindersAndSLABreaches(ctx, evalAt)
	if err != nil {
		t.Fatal(err)
	}
	if breaches1 < 1 || rem1 < 2 || esc1 < 2 {
		t.Fatalf("expected >=1 breach, >=2 reminders, >=2 escalations; got breach=%d rem=%d esc=%d", breaches1, rem1, esc1)
	}

	var siteRemOutbox, siteEscOutbox int64
	db.Model(&Outbox{}).Where("site_id = ? AND event_type = ?", siteID, "INCIDENT_ASSIGN_REMINDER").Count(&siteRemOutbox)
	db.Model(&Outbox{}).Where("site_id = ? AND event_type = ?", siteID, "INCIDENT_UNRECOVERED_ESCALATION").Count(&siteEscOutbox)
	if siteRemOutbox != 2 || siteEscOutbox != 2 {
		t.Fatalf("expected exact 1 reminder & 1 escalation per incident on site (2 total), got rem=%d esc=%d", siteRemOutbox, siteEscOutbox)
	}

	// Evaluasi ulang kedua pada waktu berikutnya tidak boleh menduplikasi event atau outbox!
	breaches2, rem2, esc2, err := svc.EvaluateRemindersAndSLABreaches(ctx, evalAt.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if breaches2 != 0 || rem2 != 0 || esc2 != 0 {
		t.Fatalf("idempotency violated on second tick: breach=%d rem=%d esc=%d", breaches2, rem2, esc2)
	}
	db.Model(&Outbox{}).Where("site_id = ? AND event_type = ?", siteID, "INCIDENT_ASSIGN_REMINDER").Count(&siteRemOutbox)
	db.Model(&Outbox{}).Where("site_id = ? AND event_type = ?", siteID, "INCIDENT_UNRECOVERED_ESCALATION").Count(&siteEscOutbox)
	if siteRemOutbox != 2 || siteEscOutbox != 2 {
		t.Fatalf("expected reminder/escalation outbox count to stay 2 after second tick, got rem=%d esc=%d", siteRemOutbox, siteEscOutbox)
	}

	db.First(&failedJob, "id = ?", failedJob.ID)
	if !failedJob.SLABreached || failedJob.SLABreachedAt == nil {
		t.Fatalf("permanently failed outbox past deadline must record sla_breached=true, got %+v", failedJob)
	}

	// Kirim ulang manual Admin (ManualRetryOutbox) mengembalikan failedJob ke pending.
	if err := svc.ManualRetryOutbox(ctx, failedJob.ID, siteID, evalAt.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var retriedJob Outbox
	db.First(&retriedJob, "id = ?", failedJob.ID)
	if retriedJob.Status != "pending" || retriedJob.FailedAt != nil {
		t.Fatalf("expected manual retry to reset status to pending, got %+v", retriedJob)
	}
}
