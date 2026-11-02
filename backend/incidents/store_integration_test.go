package incidents

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"sigap-backend/config"
	"sigap-backend/models"
	"sync"
	"testing"
	"time"
)

func integrationDB(t *testing.T) (*gorm.DB, models.CCTV) {
	t.Helper()
	dsn := os.Getenv("SIGAP_INCIDENT_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated PostgreSQL DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	// Safety: this fixture only accepts the dedicated test database.
	var name string
	db.Raw("SELECT current_database()").Scan(&name)
	if name != "sigap_incident_test" {
		t.Fatal("refusing non-test database")
	}
	if err := config.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	siteID := uuid.New()
	cameraID := uuid.New()
	site := models.Site{Base: models.Base{ID: siteID}, Kode: siteID.String()[:8], Nama: "Test Site"}
	if err := db.Create(&site).Error; err != nil {
		t.Fatal(err)
	}
	camera := models.CCTV{Base: models.Base{ID: cameraID}, KodeCCTV: cameraID.String(), QrPayload: cameraID.String(), Nama: "Kamera Uji", SiteID: &siteID, Status: "aktif"}
	if err := db.Create(&camera).Error; err != nil {
		t.Fatal(err)
	}
	return db, camera
}

func TestPostgresConcurrentOfflineRecoveryAndAdminClosure(t *testing.T) {
	db, camera := integrationDB(t)
	svc := Service{DB: db}
	ctx := context.Background()
	at := time.Now().UTC().Add(-10 * time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.Observe(ctx, camera, "offline", at); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var items []Incident
	if err := db.Preload("Members").Where("site_id = ?", camera.SiteID).Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || len(items[0].Members) != 1 {
		t.Fatalf("expected one incident/member: %+v", items)
	}
	var count int64
	db.Model(&Outbox{}).Where("incident_id = ?", items[0].ID).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate offline messages: %d", count)
	}
	if err := svc.Observe(ctx, camera, "unknown", at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := svc.Resolve(ctx, items[0].ID, *camera.SiteID, uuid.New(), "verified", at.Add(2*time.Second)); err == nil {
		t.Fatal("closure must reject unknown evidence")
	}
	for _, sec := range []int{10, 130, 250, 310} {
		if err := svc.Observe(ctx, camera, "live", at.Add(time.Duration(sec)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	var incident Incident
	db.First(&incident, "id = ?", items[0].ID)
	if incident.Status != "pending_verification" {
		t.Fatal(incident.Status)
	}
	if err := svc.Observe(ctx, camera, "offline", at.Add(320*time.Second)); err != nil {
		t.Fatal(err)
	}
	db.Model(&Incident{}).Where("site_id = ?", camera.SiteID).Count(&count)
	if count != 1 {
		t.Fatal("recurrence created duplicate ticket")
	}
	for _, sec := range []int{330, 450, 570, 630} {
		if err := svc.Observe(ctx, camera, "live", at.Add(time.Duration(sec)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	admin := uuid.New()
	adminUser := models.User{Base: models.Base{ID: admin}, Username: admin.String(), FullName: "Admin", Role: "admin", Email: admin.String() + "@test.invalid", PasswordHash: "test"}
	if err := db.Create(&adminUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Resolve(ctx, incident.ID, uuid.New(), admin, "wrong site", at.Add(631*time.Second)); err == nil {
		t.Fatal("cross-site closure accepted")
	}
	if err := svc.Resolve(ctx, incident.ID, *camera.SiteID, admin, "Verified", at.Add(631*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := svc.Observe(ctx, camera, "offline", at.Add(640*time.Second)); err != nil {
		t.Fatal(err)
	}
	db.Model(&Incident{}).Where("site_id = ?", camera.SiteID).Count(&count)
	if count != 2 {
		t.Fatal("new outage after closure needs new incident")
	}
}
