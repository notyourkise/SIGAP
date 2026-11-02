// A loopback-only presentation harness. It uses the production recovery reducer
// and Telegram sender, with an explicitly simulated clock and local HTTP receiver.
// It never loads application secrets, connects to a database, or probes cameras.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"sigap-backend/incidents"
)

//go:embed index.html
var page []byte

type rewriteTransport struct{ target *url.URL }

func (t rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	u := *r.URL
	u.Scheme = t.target.Scheme
	u.Host = t.target.Host
	clone.URL = &u
	clone.Host = t.target.Host
	return http.DefaultTransport.RoundTrip(clone)
}

type demo struct {
	mu         sync.Mutex
	Now        time.Time        `json:"now"`
	IncidentID string           `json:"incident_id"`
	Status     string           `json:"status"`
	Assigned   bool             `json:"assigned"`
	Member     incidents.Member `json:"member"`
	Messages   []string         `json:"messages"`
	Events     []string         `json:"events"`
	sender     incidents.TelegramSender
}

func (d *demo) send(text string) {
	id, err := d.sender.Send(context.Background(), "-100", text)
	if err != nil {
		d.Events = append(d.Events, "Pengiriman tiruan gagal: "+err.Error())
		return
	}
	d.Messages = append(d.Messages, fmt.Sprintf("Pesan #%d · %s", id, text))
}
func (d *demo) action(action string) error {
	if action == "reset" {
		d.Now = time.Date(2026, 11, 2, 7, 0, 0, 0, time.UTC)
		d.IncidentID = ""
		d.Status = ""
		d.Assigned = false
		d.Member = incidents.Member{}
		d.Messages = []string{}
		d.Events = []string{}
		return nil
	}
	if action == "assign" {
		if d.IncidentID == "" || d.Status == "resolved" {
			return fmt.Errorf("belum ada insiden terbuka")
		}
		if d.Assigned {
			return fmt.Errorf("teknisi sudah ditugaskan")
		}
		d.Assigned = true
		if d.Status != "pending_verification" {
			d.Status = "in_progress"
		}
		d.send("Admin menugaskan Teknisi Demo. Teknisi tidak perlu menautkan akun Telegram.")
	} else if action == "resolve" {
		if d.Status != "pending_verification" || d.Member.ConnectionState != "recovered" {
			return fmt.Errorf("belum ada bukti pemulihan stabil 5 menit")
		}
		d.Status = "resolved"
		d.Member.IsActiveMember = false
		d.send("Admin memverifikasi dan menutup insiden.")
	} else {
		state := "live"
		switch action {
		case "offline":
			state = "offline"
		case "unknown":
			state = "unknown"
		case "live":
		case "plus120":
			d.Now = d.Now.Add(120 * time.Second)
		case "plus60":
			d.Now = d.Now.Add(60 * time.Second)
		default:
			return fmt.Errorf("aksi tidak dikenal")
		}
		d.Now = d.Now.Add(time.Millisecond)
		if state == "offline" && (d.IncidentID == "" || d.Status == "resolved") {
			d.IncidentID = uuid.NewString()
			d.Status = "open"
			d.Assigned = false
			d.Member = incidents.Member{CameraName: "CCTV Gerbang · Data Demo", ConnectionState: "offline", IsActiveMember: true}
			d.send("CCTV Gerbang offline terkonfirmasi. Tiket " + d.IncidentID[:8] + " dibuat otomatis.")
		}
		if d.IncidentID != "" && d.Status != "resolved" {
			old := d.Member.ConnectionState
			recovered := incidents.Advance(&d.Member, state, d.Now)
			if recovered {
				d.Status = "pending_verification"
				d.send("CCTV stabil online 5 menit berdasarkan pemeriksaan berkala. Menunggu verifikasi admin.")
			} else if d.Member.ConnectionState != "recovered" {
				if d.Assigned {
					d.Status = "in_progress"
				} else {
					d.Status = "open"
				}
				if state == "offline" && old != "offline" {
					d.send("CCTV kembali offline. Tiket yang sama tetap digunakan.")
				}
			}
			if state == "unknown" {
				d.Events = append(d.Events, "Pemantau terganggu: kondisi belum dapat dipastikan, tanpa tiket offline baru.")
			}
		}
		d.Events = append(d.Events, d.Now.In(time.FixedZone("WITA", 8*3600)).Format("15:04:05")+" · Bukti: "+state)
	}
	return nil
}

func main() {
	var sequence int64
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"result":{"message_id":%d,"chat":{"id":-100}}}`, sequence)
	}))
	defer receiver.Close()
	target, _ := url.Parse(receiver.URL)
	d := &demo{sender: incidents.TelegramSender{Token: "local-demo-only", Client: &http.Client{Transport: rewriteTransport{target}, Timeout: time.Second}}}
	_ = d.action("reset")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)
	})
	mux.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		// Requests must originate from this loopback demo page, not arbitrary sites.
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://127.0.0.1:8095" && origin != "http://localhost:8095" {
			http.Error(w, "origin ditolak", 403)
			return
		}
		if r.Header.Get("X-SIGAP-Demo") != "1" {
			http.Error(w, "header demo wajib", 403)
			return
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		if err := d.action(r.URL.Query().Get("type")); err != nil {
			http.Error(w, err.Error(), 409)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)
	})
	log.Println("Demo SIGAP: http://127.0.0.1:8095 — kamera, waktu, dan Telegram adalah simulasi; tanpa database operasional")
	server := &http.Server{Addr: "127.0.0.1:8095", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
