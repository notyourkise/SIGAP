package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"sigap-backend/config"
	"sigap-backend/incidents"
	"sigap-backend/models"
	"sigap-backend/utils"
)

func integrationHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("SIGAP_INCIDENT_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated PostgreSQL DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	var name string
	db.Raw("SELECT current_database()").Scan(&name)
	if name != "sigap_incident_test" {
		t.Fatal("refusing non-test database")
	}
	require.NoError(t, config.RunMigrations(db))

	origDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = origDB })

	return db
}

func isolateSharedNVRPostgresCycle(t *testing.T, db *gorm.DB) {
	t.Helper()
	// Nonaktifkan kamera uji sisa paket/subtest lain di sigap_incident_test tanpa menyentuh FK maintenance_logs
	// agar runStreamVerifyCycle hanya memverifikasi kamera pada subtest aktif secara hermetic.
	var name string
	db.Raw("SELECT current_database()").Scan(&name)
	require.Equal(t, "sigap_incident_test", name)

	require.NoError(t, db.Exec("UPDATE cctvs SET status = 'nonaktif', stream_enabled = false, monitoring_enabled = false").Error)
}

func TestPostgresRunStreamVerifyCycleSharedNVRAcrossConfirmRetry(t *testing.T) {
	db := integrationHandlerDB(t)
	t.Setenv("CCTV_INCIDENTS_ENABLED", "true")

	origTimeout := streamVerifyTimeout
	origPoll := streamVerifyPollInterval
	origByteWin := streamVerifyByteWindow
	origConfirmDelay := confirmRetryDelay
	origCloseDelay := sourceCloseConfirmDelay
	origProbeNVR := probeNVRRTSPHost
	defer func() {
		streamVerifyTimeout = origTimeout
		streamVerifyPollInterval = origPoll
		streamVerifyByteWindow = origByteWin
		confirmRetryDelay = origConfirmDelay
		sourceCloseConfirmDelay = origCloseDelay
		probeNVRRTSPHost = origProbeNVR
		replaceVerifierGateForTest(nil, false)
	}()

	streamVerifyTimeout = 40 * time.Millisecond
	streamVerifyPollInterval = 10 * time.Millisecond
	streamVerifyByteWindow = 10 * time.Millisecond
	confirmRetryDelay = 25 * time.Millisecond
	sourceCloseConfirmDelay = 0

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		pathName := r.URL.Path[len("/v3/paths/get/"):]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":          pathName,
			"available":     false,
			"ready":         false,
			"tracks":        []string{},
			"bytesReceived": 0,
		})
	})
	mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
	defer func() { streamClient = oldClient }()

	encPass := encryptedTestPassword(t, "secret123")
	createSiteAndCameras := func(nvrIP string, count int) (models.Site, []models.CCTV) {
		siteID := uuid.New()
		site := models.Site{
			Base: models.Base{ID: siteID},
			Kode: "TG-" + siteID.String()[:6],
			Nama: "Site NVR Integration " + siteID.String()[:4],
		}
		require.NoError(t, db.Create(&site).Error)
		cams := make([]models.CCTV, 0, count)
		for idx := 0; idx < count; idx++ {
			id := uuid.New()
			cam := models.CCTV{
				Base:                models.Base{ID: id},
				KodeCCTV:            "NVR-" + strings.ToUpper(id.String()[:6]),
				QrPayload:           id.String(),
				Nama:                "Kamera NVR " + string(rune('A'+idx)),
				LokasiArea:          "Pos Uji NVR",
				IPNVR:               nvrIP,
				ChannelPort:         idx + 1,
				RTSPPassword:        encPass,
				SiteID:              &siteID,
				Status:              "aktif",
				StreamEnabled:       true,
				MonitoringEnabled:   true,
				StreamVerifiedState: models.StreamStateLive,
			}
			require.NoError(t, db.Create(&cam).Error)
			cams = append(cams, cam)
		}
		return site, cams
	}

	t.Run("2 kamera gagal serentak via ConfirmQueue membuka 1 tiket shared_nvr dan kamera ke-3 bergabung otomatis", func(t *testing.T) {
		isolateSharedNVRPostgresCycle(t, db)
		firstFailureMu.Lock()
		firstFailureMap = map[uuid.UUID]time.Time{}
		firstFailureMu.Unlock()

		gate := NewUnifiedNVRSessionGate(4, 2, 5*time.Millisecond, 0)
		replaceVerifierGateForTest(gate, true)
		defer replaceVerifierGateForTest(nil, false)

		nvrIP := "172.28.0.51"
		probeNVRRTSPHost = func(ip string) bool {
			assert.Equal(t, nvrIP, ip)
			return false
		}

		site, cams := createSiteAndCameras(nvrIP, 2)
		runStreamVerifyCycle(db)

		var openIncidents []incidents.Incident
		require.NoError(t, db.Preload("Members").
			Where("site_id = ? AND status NOT IN ('resolved', 'merged')", site.ID).
			Find(&openIncidents).Error)
		require.Len(t, openIncidents, 1, "must open exactly 1 shared_nvr incident instead of 2 single_camera tickets")
		inc := openIncidents[0]
		assert.Equal(t, incidents.ScopeSharedNVR, inc.ScopeType)
		assert.Equal(t, "nvr:"+nvrIP, inc.CorrelationKey)
		assert.Equal(t, incidents.StatusOpen, inc.Status)
		assert.True(t, inc.ConfirmedOfflineAt.After(inc.FirstFailedObservedAt),
			"confirmed_offline_at (%v) must remain strictly after first_failed_observed_at (%v)",
			inc.ConfirmedOfflineAt, inc.FirstFailedObservedAt)
		require.Len(t, inc.Members, 2)
		for _, m := range inc.Members {
			assert.True(t, m.IsActiveMember)
			assert.Equal(t, incidents.StateOffline, m.ConnectionState)
			require.NotNil(t, m.FirstFailedObservedAt)
			require.NotNil(t, m.ConfirmedOfflineAt)
			assert.True(t, m.ConfirmedOfflineAt.After(*m.FirstFailedObservedAt))
		}

		var outboxCount int64
		require.NoError(t, db.Model(&incidents.Outbox{}).
			Where("incident_id = ? AND event_type = 'CCTV_OFFLINE'", inc.ID).
			Count(&outboxCount).Error)
		assert.EqualValues(t, 1, outboxCount)

		// Periksa tabel master cctvs ikut diperbarui ke offline (failures=2).
		var dbCams []models.CCTV
		require.NoError(t, db.Where("id IN ?", []uuid.UUID{cams[0].ID, cams[1].ID}).Find(&dbCams).Error)
		require.Len(t, dbCams, 2)
		for _, c := range dbCams {
			assert.Equal(t, models.StreamStateOffline, c.StreamVerifiedState)
			assert.Equal(t, 2, c.StreamVerifyFailures)
		}

		// Siklus berikutnya: kamera ke-3 pada NVR mati yang sama baru gagal -> ikut bergabung ke tiket shared_nvr terbuka!
		cam3ID := uuid.New()
		cam3 := models.CCTV{
			Base:                models.Base{ID: cam3ID},
			KodeCCTV:            "NVR-" + strings.ToUpper(cam3ID.String()[:6]),
			QrPayload:           cam3ID.String(),
			Nama:                "Kamera NVR C",
			LokasiArea:          "Pos Uji NVR",
			IPNVR:               nvrIP,
			ChannelPort:         3,
			RTSPPassword:        encPass,
			SiteID:              &site.ID,
			Status:              "aktif",
			StreamEnabled:       true,
			MonitoringEnabled:   true,
			StreamVerifiedState: models.StreamStateLive,
		}
		require.NoError(t, db.Create(&cam3).Error)

		runStreamVerifyCycle(db)

		openIncidents = nil
		require.NoError(t, db.Preload("Members").
			Where("site_id = ? AND status NOT IN ('resolved', 'merged')", site.ID).
			Find(&openIncidents).Error)
		require.Len(t, openIncidents, 1, "3rd sibling camera on dead NVR must attach to existing shared_nvr ticket without creating duplicate ticket")
		assert.Equal(t, inc.ID, openIncidents[0].ID)
		assert.Equal(t, incidents.ScopeSharedNVR, openIncidents[0].ScopeType)
		assert.Contains(t, openIncidents[0].Title, "(3 CCTV terdampak)")
		require.Len(t, openIncidents[0].Members, 3)
	})

	t.Run("2 kamera gagal serentak namun NVR TCP:554 hidup jatuh ke 2 tiket single_camera terpisah", func(t *testing.T) {
		isolateSharedNVRPostgresCycle(t, db)
		firstFailureMu.Lock()
		firstFailureMap = map[uuid.UUID]time.Time{}
		firstFailureMu.Unlock()

		gate := NewUnifiedNVRSessionGate(4, 2, 5*time.Millisecond, 0)
		replaceVerifierGateForTest(gate, true)
		defer replaceVerifierGateForTest(nil, false)

		nvrIP := "172.28.0.52"
		probeNVRRTSPHost = func(ip string) bool {
			assert.Equal(t, nvrIP, ip)
			return true
		}

		site, _ := createSiteAndCameras(nvrIP, 2)
		runStreamVerifyCycle(db)

		var openIncidents []incidents.Incident
		require.NoError(t, db.Preload("Members").
			Where("site_id = ? AND status NOT IN ('resolved', 'merged')", site.ID).
			Find(&openIncidents).Error)
		require.Len(t, openIncidents, 2, "when NVR port 554 responds, each failed camera must get its own single_camera ticket")
		for _, inc := range openIncidents {
			assert.Equal(t, incidents.ScopeSingleCamera, inc.ScopeType)
			assert.Empty(t, inc.CorrelationKey)
			assert.Len(t, inc.Members, 1)
		}
	})
}
