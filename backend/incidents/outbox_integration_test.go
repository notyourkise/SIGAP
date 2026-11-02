package incidents

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPostgresOutboxPersistsTelegramReceipt(t *testing.T) {
	db, camera := integrationDB(t)
	now := time.Now().UTC()
	svc := Service{DB: db}
	if err := svc.Observe(context.Background(), camera, "offline", now); err != nil {
		t.Fatal(err)
	}
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true,"result":{"message_id":99,"chat":{"id":-100}}}`)
	}))
	defer receiver.Close()
	sender := TelegramSender{Token: "test", Client: receiver.Client(), baseURL: receiver.URL}
	chats := map[string]string{camera.SiteID.String(): "-100"}
	for i := 0; i < 100; i++ {
		worked, err := DeliverOne(context.Background(), db, sender, chats, now.Add(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if !worked {
			break
		}
	}
	var job Outbox
	if err := db.Where("site_id = ?", camera.SiteID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != "sent" || job.MessageID != 99 || job.Attempts != 1 {
		t.Fatalf("receipt not persisted: %+v", job)
	}
}

func TestPostgresRestartInvalidatesRecoveryWithoutNewTickets(t *testing.T) {
	db, camera := integrationDB(t)
	now := time.Now().UTC()
	svc := Service{DB: db}
	ctx := context.Background()
	if err := svc.Observe(ctx, camera, "offline", now); err != nil {
		t.Fatal(err)
	}
	if err := svc.Observe(ctx, camera, "live", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetObservations(ctx); err != nil {
		t.Fatal(err)
	}
	var member Member
	db.Where("cctv_id = ?", camera.ID).First(&member)
	if member.ConnectionState != "unknown" || !member.IsActiveMember || member.RecoveryStartedAt != nil {
		t.Fatalf("invalid restart state: %+v", member)
	}
	var n int64
	db.Model(&Incident{}).Where("site_id = ?", camera.SiteID).Count(&n)
	if n != 1 {
		t.Fatal("restart created duplicate incident")
	}
}
