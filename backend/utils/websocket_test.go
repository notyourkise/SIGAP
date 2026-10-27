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
