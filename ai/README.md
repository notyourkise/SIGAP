# SIGAP AI Detection Service (Task 14B, Sprint 38)

Layanan mikro Python (FastAPI) untuk deteksi orang pada kamera CCTV SIGAP.
Service ini berjalan di dalam Docker network `sigap_network` sebagai container
`sigap_ai`, membaca stream RTSP internal dari MediaMTX, melakukan deteksi orang
dengan **YOLOv8n** (~1 fps per kamera), melacak orang dengan **ByteTrack**
(ID tracking unik), menghitung jumlah orang unik per hari, lalu mengirim hasilnya
ke backend **sigap_api**.

## Alur Kerja

```
Browser /monitor
  -> frontend POST /api/ai/sessions {active_cctv_ids}
  -> sigap_api (Go)
  -> POST http://sigap_ai:8010/sessions
  -> sigap_ai (Python FastAPI)
       -> baca RTSP rtsp://mediamtx:8554/cam-{cctv_id}
       -> inferensi YOLOv8n ~1 fps + tracking ByteTrack
       -> hitung unique_today
       -> POST http://sigap_api:3000/api/ai/detection-event (header X-AI-Secret)
  -> sigap_api simpan ke DB + broadcast WebSocket AI_DETECTION_UPDATE
```

Catatan: MediaMTX menggunakan `sourceOnDemand`, sehingga stream baru aktif saat
ada viewer pertama — service AI inilah yang menjadi "viewer pertama" yang
mengaktifkan stream.

## Variabel Lingkungan (Env Vars)

| Variabel | Default | Keterangan |
| --- | --- | --- |
| `BACKEND_URL` | `http://sigap_api:3000` | URL internal backend Go untuk POST hasil deteksi |
| `MEDIAMTX_RTSP_URL` | `rtsp://mediamtx:8554` | URL RTSP internal MediaMTX (path: `cam-{cctv_id}`) |
| `AI_SECRET` | `sigap-ai-secret-change-me` | Secret header `X-AI-Secret` untuk autentikasi ke backend |

## Build & Run

```bash
# Dari root repo (docker-compose.yml sudah berisi service sigap_ai):
docker compose build sigap_ai
docker compose up -d sigap_ai

# Jalankan manual tanpa Docker (untuk pengembangan lokal):
#   python -m venv .venv && .venv/Scripts/activate  (Windows) / source .venv/bin/activate (Linux/macOS)
#   pip install -r requirements.txt
#   uvicorn main:app --host 0.0.0.0 --port 8010
```

## Endpoint

| Method | Path | Keterangan |
| --- | --- | --- |
| `GET` | `/health` | Health check (`{"status":"ok","service":"sigap_ai"}`) |
| `POST` | `/sessions` | Aktifkan sesi kamera. Body: `{"cctv_id": "...", "rtsp_url": "..."?}` (rtsp_url opsional) |
| `DELETE` | `/sessions/{cctv_id}` | Nonaktifkan sesi kamera |
| `GET` | `/sessions` | Daftar kamera aktif (`{"data": [...]}`) |
| `GET` | `/counts/{cctv_id}` | Jumlah orang unik hari ini untuk satu kamera |
| `GET` | `/counts` | Jumlah orang unik hari ini untuk semua kamera |
| `POST` | `/reset-all` | Reset manual seluruh penghitung orang unik |

## Struktur Folder

```
ai/
├── Dockerfile            # Image python:3.11-slim + OpenCV system libs
├── requirements.txt      # Dependensi Python (pin versi)
├── main.py               # Entrypoint FastAPI + scheduler reset harian (00:00 WITA)
├── core/
│   ├── detector.py       # Deteksi orang YOLOv8n (kelas 0 COCO saja)
│   ├── tracker.py        # Tracker ByteTrack (supervision)
│   ├── person_counter.py # Penghitung orang unik per hari per kamera
│   └── session_manager.py# Sesi kamera (RTSP -> inferensi -> kirim ke backend)
└── routers/
    ├── session.py        # Endpoint manajemen sesi
    └── health.py         # Endpoint health check
```

Reset harian dilakukan otomatis pukul **00:00 WITA** (UTC+8) oleh scheduler
internal pada `main.py`.
