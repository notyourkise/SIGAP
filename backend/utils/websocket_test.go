package utils

import (
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	gws "github.com/gofiber/websocket/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// resetWebSocketHub membersihkan koneksi global agar test tidak terpengaruh
// koneksi dari test lain. Koneksi client tetap ditutup oleh masing-masing test.
func resetWebSocketHub(t *testing.T) {
	t.Helper()
	Hub.mu.Lock()
	Hub.clients = make(map[*gws.Conn]WSClientInfo)
	Hub.mu.Unlock()

	t.Cleanup(func() {
		Hub.mu.Lock()
		Hub.clients = make(map[*gws.Conn]WSClientInfo)
		Hub.mu.Unlock()
	})
}

func TestWSClientInfo_CanReceiveSiteEvent(t *testing.T) {
	technician := WSClientInfo{
		UserID: "u-1",
		Role:   "teknisi",
		SiteIDs: map[string]struct{}{
			"site-a": {},
		},
	}

	assert.True(t, technician.CanReceiveSiteEvent("site-a"))
	assert.False(t, technician.CanReceiveSiteEvent("site-b"))
	assert.False(t, technician.CanReceiveSiteEvent(""))

	assert.True(t, WSClientInfo{Role: "admin"}.CanReceiveSiteEvent("site-a"))
	assert.True(t, WSClientInfo{Role: "admin"}.CanReceiveSiteEvent(""))
	assert.False(t, WSClientInfo{Role: "teknisi"}.CanReceiveSiteEvent("site-a"))
}

// TestBroadcastToSite_FiltersRecipients memastikan event site A hanya diterima
// teknisi site A dan admin, bukan teknisi site B.
func TestBroadcastToSite_FiltersRecipients(t *testing.T) {
	resetWebSocketHub(t)

	app := fiber.New()
	app.Get("/ws", gws.New(func(c *gws.Conn) {
		var sites []string
		if raw := c.Query("sites"); raw != "" {
			sites = strings.Split(raw, ",")
		}
		Hub.Register(c, c.Query("user"), c.Query("role"), sites)
		defer Hub.Unregister(c)

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	go func() { _ = fasthttp.Serve(listener, app.Handler()) }()

	dial := func(query string) *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial(
			"ws://"+listener.Addr().String()+"/ws?"+query, nil,
		)
		require.NoError(t, err)
		return conn
	}

	clientA := dial("user=teknisi-a&role=teknisi&sites=site-a")
	defer func() { _ = clientA.Close() }()
	clientB := dial("user=teknisi-b&role=teknisi&sites=site-b")
	defer func() { _ = clientB.Close() }()
	admin := dial("user=admin&role=admin")
	defer func() { _ = admin.Close() }()

	require.Eventually(t, func() bool {
		return Hub.ActiveConnections() == 3
	}, time.Second, 10*time.Millisecond)

	BroadcastToSite("site-a", "CCTV_STREAM_LOG_CREATED", map[string]string{
		"site_id":   "site-a",
		"nama":      "Kamera Site A",
		"kode_cctv": "SITE-A-01",
	})

	readEvent := func(conn *websocket.Conn) EventMessage {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
		_, raw, err := conn.ReadMessage()
		require.NoError(t, err)

		var event EventMessage
		require.NoError(t, json.Unmarshal(raw, &event))
		return event
	}

	eventA := readEvent(clientA)
	assert.Equal(t, "CCTV_STREAM_LOG_CREATED", eventA.Event)
	payloadA, ok := eventA.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "site-a", payloadA["site_id"])
	assert.Equal(t, "Kamera Site A", payloadA["nama"])

	eventAdmin := readEvent(admin)
	assert.Equal(t, "CCTV_STREAM_LOG_CREATED", eventAdmin.Event)
	payloadAdmin, ok := eventAdmin.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "site-a", payloadAdmin["site_id"])

	// Teknisi site B tidak boleh menerima event site A.
	require.NoError(t, clientB.SetReadDeadline(time.Now().Add(200*time.Millisecond)))
	_, _, err = clientB.ReadMessage()
	require.Error(t, err)

	var netErr net.Error
	assert.True(t, errors.As(err, &netErr))
	assert.True(t, netErr.Timeout())
}

func TestWSClientInfo_CanReceiveMaintenanceEvent(t *testing.T) {
	admin := WSClientInfo{UserID: "admin-1", Role: "admin"}
	techOwnerSiteA := WSClientInfo{
		UserID:  "tech-a1",
		Role:    "teknisi",
		SiteIDs: map[string]struct{}{"site-a": {}},
	}
	techOtherSiteA := WSClientInfo{
		UserID:  "tech-a2",
		Role:    "teknisi",
		SiteIDs: map[string]struct{}{"site-a": {}},
	}
	techSiteB := WSClientInfo{
		UserID:  "tech-b1",
		Role:    "teknisi",
		SiteIDs: map[string]struct{}{"site-b": {}},
	}

	assert.True(t, admin.CanReceiveMaintenanceEvent("site-a", "tech-a1"))
	assert.True(t, admin.CanReceiveMaintenanceEvent("site-a", ""))
	assert.True(t, admin.CanReceiveMaintenanceEvent("", ""))

	assert.True(t, techOwnerSiteA.CanReceiveMaintenanceEvent("site-a", "tech-a1"))
	assert.False(t, techOwnerSiteA.CanReceiveMaintenanceEvent("site-a", "tech-a2"))
	assert.False(t, techOwnerSiteA.CanReceiveMaintenanceEvent("site-a", ""))
	assert.False(t, techOwnerSiteA.CanReceiveMaintenanceEvent("site-b", "tech-a1"))

	assert.False(t, techOtherSiteA.CanReceiveMaintenanceEvent("site-a", "tech-a1"))
	assert.False(t, techSiteB.CanReceiveMaintenanceEvent("site-a", "tech-a1"))
	assert.False(t, techSiteB.CanReceiveMaintenanceEvent("site-a", "tech-b1"))
}

// H06: Reproduksi audit — koneksi teknisi site B wajib ditolak saat event pemeliharaan
// site A disiarkan, baik melalui BroadcastEvent maupun BroadcastMaintenanceEvent.
func TestAudit_MaintenanceBroadcastMustDenyOtherSite(t *testing.T) {
	resetWebSocketHub(t)
	app := fiber.New()
	registered := make(chan struct{})
	app.Get("/ws", gws.New(func(c *gws.Conn) {
		Hub.Register(c, "tech-b", "teknisi", []string{"site-b"})
		close(registered)
		defer Hub.Unregister(c)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	go func() { _ = fasthttp.Serve(listener, app.Handler()) }()

	client, _, err := websocket.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/ws", nil)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	select {
	case <-registered:
	case <-time.After(time.Second):
		t.Fatal("koneksi tidak terdaftar")
	}

	BroadcastEvent("MAINTENANCE_UPDATED", map[string]string{
		"site_id": "site-a",
		"notes":   "site A confidential work order",
	})
	require.NoError(t, client.SetReadDeadline(time.Now().Add(200*time.Millisecond)))
	_, body, err := client.ReadMessage()
	if err == nil {
		t.Errorf("teknisi site-b tidak boleh menerima event pemeliharaan site-a: %s", body)
	}
}

// H06: Menguji pengiriman event pemeliharaan pada 4 kategori koneksi sekaligus:
// 1. Admin (menerima)
// 2. Teknisi pemilik di site yang sama (menerima)
// 3. Teknisi lain di site yang sama (ditolak)
// 4. Teknisi di site lain (ditolak)
func TestBroadcastMaintenanceEvent_SiteAndOwnershipBoundaries(t *testing.T) {
	resetWebSocketHub(t)

	app := fiber.New()
	app.Get("/ws", gws.New(func(c *gws.Conn) {
		var sites []string
		if raw := c.Query("sites"); raw != "" {
			sites = strings.Split(raw, ",")
		}
		Hub.Register(c, c.Query("user"), c.Query("role"), sites)
		defer Hub.Unregister(c)

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	go func() { _ = fasthttp.Serve(listener, app.Handler()) }()

	dial := func(query string) *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial(
			"ws://"+listener.Addr().String()+"/ws?"+query, nil,
		)
		require.NoError(t, err)
		return conn
	}

	adminConn := dial("user=admin-1&role=admin")
	defer func() { _ = adminConn.Close() }()
	ownerSiteAConn := dial("user=tech-a1&role=teknisi&sites=site-a")
	defer func() { _ = ownerSiteAConn.Close() }()
	otherTechSiteAConn := dial("user=tech-a2&role=teknisi&sites=site-a")
	defer func() { _ = otherTechSiteAConn.Close() }()
	techSiteBConn := dial("user=tech-b1&role=teknisi&sites=site-b")
	defer func() { _ = techSiteBConn.Close() }()

	require.Eventually(t, func() bool {
		return Hub.ActiveConnections() == 4
	}, time.Second, 10*time.Millisecond)

	BroadcastMaintenanceEvent("site-a", "tech-a1", "MAINTENANCE_UPDATED", map[string]string{
		"id":            "log-1",
		"site_id":       "site-a",
		"technician_id": "tech-a1",
		"status":        "ongoing",
	})

	readEvent := func(conn *websocket.Conn) EventMessage {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
		_, raw, err := conn.ReadMessage()
		require.NoError(t, err)
		var event EventMessage
		require.NoError(t, json.Unmarshal(raw, &event))
		return event
	}

	assertNoEvent := func(conn *websocket.Conn, label string) {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(200*time.Millisecond)))
		_, raw, err := conn.ReadMessage()
		require.Error(t, err, "%s tidak boleh menerima event pemeliharaan milik teknisi lain: %s", label, string(raw))
		var netErr net.Error
		assert.True(t, errors.As(err, &netErr))
		assert.True(t, netErr.Timeout())
	}

	adminMsg := readEvent(adminConn)
	assert.Equal(t, "MAINTENANCE_UPDATED", adminMsg.Event)

	ownerMsg := readEvent(ownerSiteAConn)
	assert.Equal(t, "MAINTENANCE_UPDATED", ownerMsg.Event)

	assertNoEvent(otherTechSiteAConn, "teknisi lain dalam site-a")
	assertNoEvent(techSiteBConn, "teknisi site-b")
}
