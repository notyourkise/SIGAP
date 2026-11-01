package utils

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gofiber/websocket/v2"
)

// EventMessage mendefinisikan struktur pesan yang dikirimkan via WebSocket
type EventMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// WSClientInfo menyimpan identitas pengguna di balik sebuah koneksi WebSocket.
// Dipakai untuk broadcast terarah per-user (BroadcastToUser), per-role
// (BroadcastToRole), dan per-site (BroadcastToSite) — prasyarat arsitektur
// notifikasi in-app (Fase 5.1) sekaligus isolasi data multi-site.
type WSClientInfo struct {
	UserID string
	Role   string
	// SiteIDs berisi seluruh site yang ditugaskan ke pengguna (user_sites).
	// Admin tidak memerlukannya karena memiliki akses ke semua site.
	SiteIDs map[string]struct{}
}

// CanReceiveSiteEvent menentukan apakah koneksi berhak menerima event dari
// site tertentu. Admin selalu boleh; non-admin hanya boleh untuk site yang
// ditugaskan. Event tanpa site (siteID kosong) hanya dikirim ke admin agar
// data yang belum terklasifikasi tidak bocor lintas site.
func (i WSClientInfo) CanReceiveSiteEvent(siteID string) bool {
	if i.Role == "admin" {
		return true
	}
	if siteID == "" {
		return false
	}
	_, ok := i.SiteIDs[siteID]
	return ok
}

// CanReceiveMaintenanceEvent menentukan apakah koneksi berhak menerima event
// pemeliharaan (maintenance log). Admin selalu berhak menerima semua event;
// non-admin hanya berhak bila terdaftar pada site terkait DAN merupakan
// teknisi pemilik tugas (technicianID cocok dengan UserID).
func (i WSClientInfo) CanReceiveMaintenanceEvent(siteID string, technicianID string) bool {
	if i.Role == "admin" {
		return true
	}
	if !i.CanReceiveSiteEvent(siteID) {
		return false
	}
	if technicianID == "" || i.UserID == "" {
		return false
	}
	return i.UserID == technicianID
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
// pengguna yang terautentikasi (userID & role berasal dari klaim JWT) dan
// daftar site yang ditugaskan. SiteIDs diabaikan untuk admin yang memang
// memiliki akses global.
func (h *WSHubStore) Register(conn *websocket.Conn, userID string, role string, siteIDs []string) {
	sites := make(map[string]struct{}, len(siteIDs))
	for _, siteID := range siteIDs {
		if siteID != "" {
			sites[siteID] = struct{}{}
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = WSClientInfo{UserID: userID, Role: role, SiteIDs: sites}
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

// ActiveConnections mengembalikan jumlah koneksi WebSocket aktif saat ini
// (thread-safe). Dipakai oleh metrik `sigap_ws_active_connections` (Task 7.4).
func (h *WSHubStore) ActiveConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func extractBroadcastScope(eventType string, payload interface{}) (siteID string, technicianID string, isMaintenance bool) {
	upperEvent := strings.ToUpper(strings.TrimSpace(eventType))
	if strings.HasPrefix(upperEvent, "MAINTENANCE_") || strings.HasPrefix(upperEvent, "OVERDUE_") {
		isMaintenance = true
	}

	switch v := payload.(type) {
	case map[string]string:
		siteID = strings.TrimSpace(v["site_id"])
		if tech, ok := v["technician_id"]; ok {
			technicianID = strings.TrimSpace(tech)
			isMaintenance = true
		}
	case map[string]interface{}:
		if rawSite, ok := v["site_id"]; ok && rawSite != nil {
			siteID = strings.TrimSpace(fmt.Sprint(rawSite))
		}
		if rawTech, ok := v["technician_id"]; ok {
			isMaintenance = true
			if rawTech != nil {
				technicianID = strings.TrimSpace(fmt.Sprint(rawTech))
			}
		}
	default:
		if payload != nil {
			if raw, err := json.Marshal(payload); err == nil {
				var m map[string]interface{}
				if err := json.Unmarshal(raw, &m); err == nil {
					if rawSite, ok := m["site_id"]; ok && rawSite != nil {
						siteID = strings.TrimSpace(fmt.Sprint(rawSite))
					} else if cctvMap, ok := m["cctv"].(map[string]interface{}); ok {
						if nestedSite, ok := cctvMap["site_id"]; ok && nestedSite != nil {
							siteID = strings.TrimSpace(fmt.Sprint(nestedSite))
						}
					}
					if rawTech, ok := m["technician_id"]; ok {
						isMaintenance = true
						if rawTech != nil {
							technicianID = strings.TrimSpace(fmt.Sprint(rawTech))
						}
					}
				}
			}
		}
	}
	if siteID == "<nil>" || siteID == "00000000-0000-0000-0000-000000000000" {
		siteID = ""
	}
	if technicianID == "<nil>" || technicianID == "00000000-0000-0000-0000-000000000000" {
		technicianID = ""
	}
	return siteID, technicianID, isMaintenance
}

// BroadcastEvent mempublikasikan event notifikasi JSON dengan menerapkan
// penyaringan site dan ownership secara defensif agar event pemeliharaan maupun
// event bertarget site tidak pernah bocor lintas site atau antar-teknisi.
func BroadcastEvent(eventType string, payload interface{}) {
	siteID, technicianID, isMaintenance := extractBroadcastScope(eventType, payload)
	if isMaintenance {
		BroadcastMaintenanceEvent(siteID, technicianID, eventType, payload)
		return
	}
	BroadcastToSite(siteID, eventType, payload)
}

// BroadcastMaintenanceEvent mempublikasikan event pemeliharaan hanya kepada
// koneksi admin dan teknisi pemilik tugas yang berada pada site terkait.
func BroadcastMaintenanceEvent(siteID string, technicianID string, eventType string, payload interface{}) {
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
		if !info.CanReceiveMaintenanceEvent(siteID, technicianID) {
			continue
		}
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

// BroadcastToSite mempublikasikan event hanya ke koneksi admin dan koneksi
// user yang ditugaskan ke site terkait. Ini mencegah event CCTV site A bocor
// ke teknisi site B. Event dengan siteID kosong hanya dikirim ke admin.
func BroadcastToSite(siteID string, eventType string, payload interface{}) {
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
		if !info.CanReceiveSiteEvent(siteID) {
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
