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

// WSClientInfo menyimpan identitas pengguna di balik sebuah koneksi WebSocket.
// Dipakai untuk broadcast terarah per-user (BroadcastToUser) dan per-role
// (BroadcastToRole) — prasyarat arsitektur notifikasi in-app (Fase 5.1).
type WSClientInfo struct {
	UserID string
	Role   string
}

// WSHubStore mengelola seluruh koneksi WebSocket aktif secara thread-safe
type WSHubStore struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]WSClientInfo
}

var Hub = &WSHubStore{
	clients: make(map[*websocket.Conn]WSClientInfo),
}

// Register mendaftarkan koneksi WebSocket client baru ke Hub beserta identitas
// pengguna yang terautentikasi (userID & role berasal dari klaim JWT).
func (h *WSHubStore) Register(conn *websocket.Conn, userID string, role string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = WSClientInfo{UserID: userID, Role: role}
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

// BroadcastToUser mempublikasikan event hanya ke koneksi milik user tertentu.
// Aman dipanggil meskipun user belum membuka koneksi (guard clause tanpa client).
func BroadcastToUser(userID string, eventType string, payload interface{}) {
	Hub.mu.RLock()
	defer Hub.mu.RUnlock()

	if len(Hub.clients) == 0 {
		return
	}

	msg := EventMessage{
		Event:   eventType,
		Payload: payload,
	}

	for client, info := range Hub.clients {
		if info.UserID != userID {
			continue
		}
		err := client.WriteJSON(msg)
		if err != nil {
			Log.Warnf("⚠️ Gagal mengirim pesan WebSocket ke %s: %v", client.RemoteAddr(), err)
			go Hub.Unregister(client)
		}
	}
}

// BroadcastToRole mempublikasikan event ke seluruh koneksi user dengan peran
// tertentu (mis. semua admin). User tanpa koneksi aktif tidak menerima apa pun
// — notifikasi tetap tersimpan di database untuk diambil saat membuka bell.
func BroadcastToRole(role string, eventType string, payload interface{}) {
	Hub.mu.RLock()
	defer Hub.mu.RUnlock()

	if len(Hub.clients) == 0 {
		return
	}

	msg := EventMessage{
		Event:   eventType,
		Payload: payload,
	}

	for client, info := range Hub.clients {
		if info.Role != role {
			continue
		}
		err := client.WriteJSON(msg)
		if err != nil {
			Log.Warnf("⚠️ Gagal mengirim pesan WebSocket ke %s: %v", client.RemoteAddr(), err)
			go Hub.Unregister(client)
		}
	}
}
