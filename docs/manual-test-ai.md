# Uji Manual — AI Microservice & Person Counting (Task 14B)

> **Task:** 14B.1 + 14B.2 + 14B.3 AI Microservice & Person Counting
> **Branch:** `feature/ai-detection-service`
> **Versi:** backend `0.61.0` (Sprint 38) · service `sigap_ai` (container baru)
> **Dokumen:** verifikasi service deteksi orang (YOLOv8n + ByteTrack) pada service `sigap_ai`, integrasi backend Go (migrasi `000020`/`000021`, endpoint internal `POST /api/ai/detection-event` dengan `X-AI-Secret`, proxy session `/api/ai/sessions`, ringkasan `/api/ai/counts`), serta alur `sourceOnDemand` MediaMTX yang mengaktifkan stream saat AI mulai pull RTSP.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Build service AI | `cd ai && docker build -t sigap_ai .` (download model `yolov8n.pt` saat build; butuh internet) |
| P2 | Stack berjalan | `docker compose up -d --build` — pastikan container `sigap_ai`, `sigap_api`, `sigap_mediamtx` berstatus Up/healthy |
| P3 | Database dimigrasi | Migration `000020_ai_detection_sessions` & `000021_ai_detection_events` terjalankan (tabel `ai_detection_sessions` & `ai_detection_events` ada) |
| P4 | Shared secret | `AI_SECRET` sama di root `.env` (docker-compose) & `backend/.env.docker` (fallback `sigap-ai-secret-change-me`) |
| P5 | CCTV dengan streaming aktif | Minimal 1–2 CCTV `stream_enabled=true` + `channel_port` terisi, dan stream sudah terdaftar di MediaMTX (`GET /api/streams/status`) |
| P6 | Kamera menghadap area ramai | Arahkan salah satu kamera ke area yang dilalui orang (untuk S2/S3) |
| P7 | Akun admin | Login sebagai **admin** (endpoint AI admin-only & halaman `/monitor`) |
| P8 | Browser + DevTools | Chrome/Edge; tab Network untuk cek request `POST /api/ai/sessions`, tab WS untuk event `AI_DETECTION_UPDATE` |

## 2. Pengujian Otomatis

```bash
# Build AI service
cd ai
docker build -t sigap_ai .

# Test imports (tanpa runtime kamera)
docker run --rm sigap_ai python -c "from core.detector import Detector; from core.tracker import PersonTracker; from core.person_counter import PersonCounter; print('OK')"

# Backend
cd ../backend
go build ./... && go test ./... -count=1 -run "TestReceiveDetectionEvent|TestGetAICounts|TestGetAICountByCCTV" && go vet ./...

# Integration (service reachable)
docker compose up -d sigap_ai sigap_api mediamtx
curl http://sigap_ai:8010/health   # dari dalam network; dari host gunakan docker compose exec
curl http://localhost:8010/health   # bila port dipublish untuk pengujian
```

Hasil yang diharapkan: `docker build` sukses (exit 0), test imports mencetak `OK`, `go build`/`go test`/`go vet` lulus, dan `/health` service AI mengembalikan `{"status":"ok","service":"sigap_ai"}`.

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Migrasi & tabel database AI

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka DB (`psql`/pgAdmin) atau `\d ai_detection_sessions` | Tabel `ai_detection_sessions` ada: `id uuid PK`, `cctv_id uuid FK`, `session_date date`, `unique_person_count int`, `last_tracking_ids jsonb`, UNIQUE `(cctv_id, session_date)` | |
| 2 | Cek tabel `ai_detection_events` | Tabel `ai_detection_events` ada: `event_type varchar(50)`, `person_count`, `unique_count`, `confidence_avg`, `detected_at` + index `(cctv_id, detected_at DESC)` | |
| 3 | Cek nilai default | `unique_person_count` default 0; `event_type` default `'person_detected'` | |

### S2 - Service `sigap_ai` berjalan & health check

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | `docker compose ps` | Container `sigap_ai` berstatus `Up` (restart unless-stopped) | |
| 2 | `curl http://sigap_ai:8010/health` (dari dalam network) | Respons `{"status":"ok","service":"sigap_ai"}` (HTTP 200) | |
| 3 | Lihat log `sigap_ai` | Tidak ada error fatal saat startup; model YOLOv8n berhasil dimuat | |
| 4 | `docker compose config --quiet` | Konfigurasi compose valid (service `sigap_ai` dengan resource limits 2 CPU/2048M) | |

### S3 - Aktivasi session via proxy backend (POST /api/ai/sessions)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **admin**, panggil `POST /api/ai/sessions` dengan body `{"active_cctv_ids":["<uuid>"]}` | Respons `200` dengan `activated` berisi UUID CCTV; service AI mulai pull RTSP (log `✅ AI Session aktif: <uuid>`) | |
| 2 | Cek log service AI | AI service melakukan `cv2.VideoCapture` ke `rtsp://mediamtx:8554/cam-<uuid>`; karena `sourceOnDemand: yes`, MediaMTX mulai menarik stream dari NVR (stream aktif saat AI menjadi viewer pertama) | |
| 3 | Cek `GET /api/ai/sessions` (admin) | Respons menampilkan daftar session aktif + `unique_today` | |
| 4 | Nonaktifkan: `DELETE /api/ai/sessions/<uuid>` | Session berhenti (log `⏹ AI Session berhenti`); CPU service AI turun; `GET /api/ai/sessions` tidak lagi memuat kamera tersebut | |

### S4 - Deteksi orang & bounding box (end-to-end)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Aktifkan AI untuk kamera yang menghadap area ramai | Buka `/monitor` (frontend Sprint 39) atau periksa WebSocket; bounding box hijau muncul di atas video saat orang terdeteksi | |
| 2 | Arahkan orang berjalan melewati kamera | `ai_detection_events` terisi baris baru (`person_count > 0`, `confidence_avg` terisi) — cek via DB | |
| 3 | Cek event WebSocket `AI_DETECTION_UPDATE` | Payload berisi `cctv_id`, `bboxes` (koordinat relatif 0–1), `unique_today`, `current_in_frame` | |
| 4 | Cek `POST /api/ai/detection-event` langsung (dengan `X-AI-Secret` benar) | Respons `{"status":"success"}` (HTTP 200) | |
| 5 | Cek `POST /api/ai/detection-event` tanpa/salah `X-AI-Secret` | Respons `401 Unauthorized` | |

### S5 - Unique person counting (orang sama tidak dihitung dua kali)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Satu orang berjalan masuk → keluar → masuk lagi frame | `unique_today` di `ai_detection_sessions` tetap 1 (tidak bertambah) karena tracking ID sama | |
| 2 | Orang kedua lewat | `unique_today` bertambah menjadi 2 | |
| 3 | Cek `GET /api/ai/counts` (admin) | Ringkasan per kamera menampilkan `nama`, `kode_cctv`, `unique_today` sesuai DB | |
| 4 | Cek `GET /api/ai/counts/<uuid>` (admin) | Detail per kamera menampilkan `unique_today` hari ini | |

### S6 - Counter reset 00:00 WITA

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Tunggu atau panggil `POST /reset-all` pada service AI | Log `🔄 Daily reset: person counter semua kamera direset (00:00 WITA)`; `GET /counts` semua kembali 0 | |
| 2 | Setelah reset, hari baru di DB | `ai_detection_sessions` tetap menyimpan data hari kemarin; `session_date` baru dibuat terpisah saat deteksi berikutnya | |

### S7 - MediaMTX sourceOnDemand & AI sebagai viewer pertama

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan tidak ada viewer WHEP yang membuka stream | MediaMTX path `cam-<uuid>` belum `ready` (cek `GET /api/streams/status`) | |
| 2 | Aktifkan AI session untuk kamera tersebut | MediaMTX mulai menarik RTSP dari NVR (path menjadi `ready`) karena AI melakukan pull — tanpa perlu membuka browser | |
| 3 | Stop AI session & tunggu `sourceOnDemandCloseAfter: 30s` | Path MediaMTX kembali non-ready (stream ditutup otomatis) | |

### S8 - Role enforcement

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi**, akses `GET /api/ai/counts` | Respons **403 Forbidden** (endpoint hanya admin) | |
| 2 | Teknisi akses `POST /api/ai/sessions` | Respons **200** (proxy session boleh semua role login — frontend teknisi dapat mengaktifkan AI di monitor bila diizinkan aplikasi) | |
| 3 | Tanpa sesi, akses `GET /api/ai/counts` | Respons **401 Unauthorized** | |
| 4 | Tanpa sesi, akses `POST /api/ai/detection-event` dengan `X-AI-Secret` benar | Respons **200** (endpoint internal tidak butuh JWT — hanya shared secret) | |

### S9 - Kegagalan layanan AI (degradasi graceful)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Hentikan container `sigap_ai` (`docker compose stop sigap_ai`) | Backend `sigap_api` tetap berjalan normal; endpoint AI `GET /api/ai/counts` tetap 200 dengan data DB (cache sesi) | |
| 2 | Panggil `POST /api/ai/sessions` saat AI down | Respons `502 Bad Gateway` (`failed` berisi `Gagal menghubungi service AI`) — tidak crash backend | |
| 3 | Nyalakan kembali `sigap_ai` | `GET /health` kembali 200; session dapat diaktifkan kembali | |

## 4. Kriteria Lulus

1. `docker build -t sigap_ai .` sukses tanpa error; service `sigap_ai` start dan `/health` mengembalikan 200.
2. Migration `000020`/`000021` membuat tabel `ai_detection_sessions` & `ai_detection_events` sesuai skema.
3. `POST /api/ai/sessions` (proxy) mengaktifkan tracking untuk kamera yang diminta; `DELETE /api/ai/sessions/{id}` menghentikan tracking & membebaskan resource.
4. Person yang sama tidak dihitung dua kali dalam satu hari (tracking ID unik via ByteTrack).
5. Counter di-reset ke 0 pada 00:00 WITA (scheduler service AI) atau `POST /reset-all`.
6. Backend menerima event dari AI service (`POST /api/ai/detection-event` + `X-AI-Secret`) dan broadcast WebSocket `AI_DETECTION_UPDATE`.
7. `GET /api/ai/counts` mengembalikan data hari ini; endpoint admin-only (teknisi → 403, tanpa sesi → 401).
8. Backend tidak crash saat service AI down (degradasi graceful, proxy → 502).
9. `go build`/`go test`/`go vet` lulus (termasuk `handlers/ai_test.go`).

## Catatan

- **YOLOv8n** hanya mendeteksi class 0 (`person`) dengan confidence ≥ 0.50; inference ±1 frame/detik per kamera (throttle) agar CPU efisien.
- **MediaMTX `sourceOnDemand: yes`**: stream RTSP dari NVR baru ditarik saat ada yang pull — AI service adalah "viewer pertama" yang mengaktifkan stream saat session diaktifkan; setelah AI berhenti, stream ditutup otomatis (`sourceOnDemandCloseAfter: 30s`).
- **RTSP internal AI**: `rtsp://mediamtx:8554/cam-{cctv_id}` — path mengikuti `BuildStreamPath()` di `utils/rtsp.go`.
- **Shared secret `X-AI-Secret`**: WAJIB sama antara `AI_SECRET` di root `.env` dan `backend/.env.docker`; ganti dari nilai default sebelum produksi.
- **Frontend**: overlay bounding box & tombol AI di `/monitor` adalah Sprint 39 (14C) — skenario S4/S5 di atas dapat diverifikasi sebagian lewat WebSocket/DB hingga frontend selesai.
- **Port 8010 tidak dipublish** di docker-compose; akses via `docker compose exec sigap_ai curl ...` atau jalankan `docker run --rm --network sigap_network sigap_ai ...` untuk pengujian internal.
- Endpoint terkait: `POST /api/ai/detection-event` (internal) · `GET /api/ai/counts` & `/api/ai/counts/:id` (admin) · `POST/DELETE/GET /api/ai/sessions` (proxy) · service `sigap_ai` (`/sessions`, `/health`, `/counts`, `/reset-all`).
