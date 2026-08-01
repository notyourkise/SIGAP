package utils

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

// EventMessage mendefinisikan struktur pesan yang dikirimkan via WebSocket
type EventMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// WSHubStore mengelola seluruh koneksi WebSocket aktif secara thread-safe
type WSHubStore struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

var Hub = &WSHubStore{
	clients: make(map[*websocket.Conn]bool),
}

// Register mendaftarkan koneksi WebSocket client baru ke Hub
func (h *WSHubStore) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	Log.Infof("🔌 Client WebSocket terhubung: %s (Total aktif: %d)", conn.RemoteAddr(), len(h.clients))
}

// Unregister menghapus koneksi WebSocket client dari Hub dan menutup koneksi
func (h *WSHubStore) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[conn]; ok {
		delete(h.clients, conn)
		_ = conn.Close()
		Log.Infof("🔌 Client WebSocket terputus: %s (Total aktif: %d)", conn.RemoteAddr(), len(h.clients))
	}
}

// BroadcastEvent mempublikasikan event notifikasi JSON ke seluruh client WebSocket yang terhubung
func BroadcastEvent(eventType string, payload interface{}) {
	Hub.mu.RLock()
	defer Hub.mu.RUnlock()

	if len(Hub.clients) == 0 {
		return
	}

	msg := EventMessage{
		Event:   eventType,
		Payload: payload,
	}

	for client := range Hub.clients {
		err := client.WriteJSON(msg)
		if err != nil {
			Log.Warnf("⚠️ Gagal mengirim pesan WebSocket ke %s: %v", client.RemoteAddr(), err)
			go Hub.Unregister(client)
		}
	}
}
