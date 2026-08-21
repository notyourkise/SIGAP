# Production Artifact Provenance — 20 Agustus 2026

Status: **✅ COMPLETE — evidence host dan permukaan publik tersedia**

Dokumen ini menjadi evidence Task 0.1 pada `docs/plans/FINDINGS-REMEDIATION-PLAN.md`. Pengambilan dilakukan read-only. Tidak ada token, password, private key, URL RTSP berkredensial, atau nilai environment rahasia yang dicatat.

## Target dan Sumber Pembanding

| Field | Nilai |
|---|---|
| Target publik | `https://nexaspace.site` |
| Host production | `76.13.19.249` / `srv1793647` |
| Environment runtime | `APP_ENV=production` |
| Waktu evidence publik | 2026-08-20 09:03:49–09:03:50 WITA |
| Waktu evidence host | 2026-08-20 09:18:59–09:21 WITA |
| Deployment checkout | `/opt/sigap`, branch `main`, commit `38e3260710ec4b2a1cd2a6f29b8786cb3b0c77b2` |
| Commit time deployment checkout | 2026-08-14T16:53:55+08:00 |
| Source pembanding | branch `main`, commit `5394ea0c543766aa9979115f899307076152fd12` |
| Jarak commit | Production checkout tertinggal 2 commit; keduanya perubahan manual-test/audit dokumentasi (#111 dan #112). |
| Operator | Anto, melalui endpoint publik dan SSH read-only |

## Topologi Production Terverifikasi

```text
Internet :80/:443
└── Nginx host 1.24.0 (Ubuntu)
    └── 127.0.0.1:8080
        └── sigap_nginx 1.27.5 (container nginx:1.27-alpine)
            ├── sigap_frontend
            ├── sigap_api
            └── sigap_mediamtx (WHEP/HLS)

Internet UDP :8189
└── docker-proxy
    └── sigap_mediamtx :8189/udp
```

Header publik `nginx/1.24.0 (Ubuntu)` bukan drift versi container. Header berasal dari Nginx host yang menjadi edge di depan container Nginx.

## Artifact Runtime

Waktu `started` dipakai sebagai marker runtime deployment karena host tidak memiliki deployment event log yang mengikat image ke satu transaksi rilis.

| Komponen | Versi/runtime | Image digest atau immutable ID | Image created | Container started (WITA) | Port host |
|---|---|---|---|---|---|
| Frontend | Node `20.20.2`; Next.js `16.2.11` | `sigap-frontend@sha256:9e713257b6ef62353b6a4e360d8e5a2d4edded2dfca686e96ad7505573bcee25` | 2026-08-15 19:07:29 WITA | 2026-08-15 19:52:47 WITA | Tidak dipublish |
| API | Binary SHA-256 `39d45439b2c23e2145731e932ce7c52c8721d98f4b4970dc111b3d1527131fb3` | `sigap-api@sha256:3f1b32f09c1f8eb2001464bd10bae7c0cd117f93bc37a891fad7235477f5376e` | 2026-08-15 18:27:45 WITA | 2026-08-15 19:08:12 WITA | Tidak dipublish |
| Container Nginx | Nginx `1.27.5` | `nginx@sha256:65645c7bb6a0661892a8b03b89d0743208a18dd2f3f17a54ef4b76fb8e2f2a10` | 2025-04-16 22:50:31 WITA | 2026-08-15 18:30:18 WITA | `127.0.0.1:8080 → 80/tcp` |
| MediaMTX | `v1.20.0` | `bluenviron/mediamtx@sha256:4be33da5a77ca41d9052b58a8fe4708c4908d901bd604253fa56dd3257833809` | 2026-08-05 18:34:47 WITA | 2026-08-15 18:30:17 WITA | `0.0.0.0/[::]:8189 → 8189/udp` |
| Edge Nginx host | Nginx `1.24.0 (Ubuntu)` | Paket host; bukan image Docker | Tidak tersedia | Service aktif saat evidence | Publik `80/tcp` dan `443/tcp` |

## Konfigurasi Runtime Non-secret

| Komponen | Evidence konfigurasi |
|---|---|
| Frontend | Nama variable tersedia: `INTERNAL_BACKEND_URL`, `NODE_ENV`, `NODE_VERSION`, `PATH`, `YARN_VERSION`. |
| API | `APP_ENV=production`. Nama variable tersedia untuk DB, Redis, JWT/AES, VAPID, SMTP, MediaMTX, RTSP NVR, CORS, trusted proxy, timezone, pool DB, default admin, dan logging. Nilai selain `APP_ENV` tidak dicatat. |
| Container Nginx | Image `nginx:1.27-alpine`; hanya variable bawaan image. Edge host meneruskan domain production ke `127.0.0.1:8080`. |
| MediaMTX | `MTX_PROTOCOLS` tersedia; config host `/opt/sigap/mediamtx.yml` di-mount read-only ke `/mediamtx.yml`. |

Nama environment API yang terverifikasi tersedia: `ADMIN_DEFAULT_PASSWORD`, `AES_SECRET_KEY`, `APP_ENV`, `CORS_ALLOWED_ORIGINS`, `DB_*`, `JWT_SECRET`, `LOG_LEVEL`, `MEDIAMTX_API_URL`, `REDIS_*`, `RTSP_NVR_*`, `SMTP_*`, `TRUSTED_PROXY_IPS`, `TZ`, dan `VAPID_*`. Daftar ini hanya membuktikan presence, bukan validitas nilainya.

## ICE dan MediaMTX

Konfigurasi aktif:

```yaml
webrtcAddress: :8889
webrtcLocalUDPAddress: :8189
webrtcICEServers2:
  - url: stun:stun.l.google.com:19302
```

Fakta runtime:

- UDP `8189` dipublish ke IPv4 dan IPv6 host.
- Tidak ada `webrtcAdditionalHosts` pada konfigurasi aktif.
- Route publik `/stream/` mencapai upstream dan mengembalikan `404 text/plain` untuk path kosong.
- Belum ada evidence WHEP signaling, ICE connected, atau frame playback untuk path kamera nyata.

Ketiadaan `webrtcAdditionalHosts` adalah kandidat akar masalah BB-08 pada NAT/jaringan eksternal, tetapi baru boleh dikonfirmasi melalui test WHEP/ICE setelah konfigurasi diperbaiki.

## Evidence Permukaan Publik

| Komponen | Pemeriksaan | Hasil | Batas bukti |
|---|---|---|---|
| Frontend | `GET /` | `307` menuju `/login`; `X-Powered-By: Next.js`. | Tidak mengungkap revision artifact. |
| Monitor | `GET /monitor` tanpa sesi | `200`, HTML Next.js, cache HIT. | Tidak membuktikan authorization setelah hydration atau playback. |
| API | `GET /api/health` | `200`; API, PostgreSQL, Redis, dan uploads sehat pada `2026-08-20T09:03:49+08:00`. | Tidak mengembalikan build SHA atau image ID. |
| Edge | Header response | `Server: nginx/1.24.0 (Ubuntu)`. | Terbukti berasal dari edge host, bukan container Nginx. |
| MediaMTX route | `GET /stream/` | `404 text/plain`, CORS `*`. | Membuktikan routing, bukan kesiapan stream. |

## Dirty Checkout dan Traceability Gap

Checkout production berada di commit `38e3260`, tetapi tidak bersih:

| File | Kondisi | Waktu modifikasi WITA | Working tree SHA-256 berbeda dari HEAD |
|---|---|---|---|
| `docker-compose.yml` | 3 baris tambah, 2 hapus | 2026-08-15 18:24:20 | Ya |
| `frontend/public/sw.js` | 12 tambah, 1 hapus | 2026-08-15 18:49:54 | Ya |
| `frontend/src/app/page.tsx` | 8 tambah, 2 hapus | 2026-08-15 19:06:38 | Ya |
| `nginx/nginx.conf` | 17 tambah, 49 hapus | 2026-08-15 18:24:20 | Ya |

Terdapat pula tiga file `.bak` yang tidak dilacak. `mediamtx.yml` bersih dan identik dengan HEAD deployment.

Image frontend dan API tidak memiliki OCI revision label/build SHA. Karena build lokal terjadi saat checkout memiliki modifikasi, commit `38e3260` adalah checkout provenance, bukan bukti pasti source tree artifact. Ini adalah gap release engineering yang wajib diperbaiki pada deployment berikutnya dengan:

- image tag immutable per commit;
- label `org.opencontainers.image.revision`;
- endpoint/version metadata frontend dan API;
- deployment hanya dari clean checkout;
- pencatatan waktu deploy dan digest pada release log.

## Matriks Production Artifact vs Source

| Komponen | Production vs source `main` `5394ea0` | Status |
|---|---|---|
| Frontend | Versi dependency cocok dengan source, tetapi artifact dibangun dari dirty deployment checkout dan tidak memiliki revision label. Dua commit setelah checkout hanya dokumentasi, namun modifikasi lokal frontend material. | **MISMATCH / SOURCE SHA UNTRACEABLE** |
| API | Tidak ada file API tracked yang dirty dan checkout hanya tertinggal dua commit dokumentasi. Binary tidak membawa revision label sehingga kesetaraan byte-ke-source tidak dapat dibuktikan. | **MATCH CANDIDATE / UNPROVABLE** |
| Container Nginx | Image runtime sesuai deklarasi `nginx:1.27-alpine`, tetapi config yang di-mount berbeda dari HEAD/source. Edge Nginx host merupakan layer tambahan yang tidak dideklarasikan di compose repository. | **MISMATCH CONFIG/TOPOLOGY** |
| MediaMTX | Config aktif identik dengan HEAD deployment dan source pembanding; digest runtime tercatat. Source masih memakai mutable tag `latest-ffmpeg`, sehingga rebuild berikutnya tidak reproducible. | **MATCH CURRENT ARTIFACT / DECLARATION NON-REPRODUCIBLE** |

## Dampak terhadap Temuan

- **BB-08 / streaming:** MediaMTX hidup dan routing tersedia, tetapi tidak ada public additional host maupun playback evidence. Temuan tetap terbuka.
- **BB-13 / maintenance 9/9:** payload production harus ditautkan ke image API digest di atas, bukan hanya checkout SHA.
- **BB-16 / laporan kustom dan BB-20 / suku cadang:** source guard tidak cukup sebagai bukti; E2E wajib dijalankan pada frontend/API digest di atas.
- **BB-25 / tombol SLA:** dirty frontend artifact menjelaskan mengapa source/changelog dan UI production dapat berbeda.

## Exit Criteria Task 0.1

- [x] Source SHA pembanding dan deployment checkout SHA dicatat.
- [x] Evidence publik dicatat dengan timestamp WITA.
- [x] Image digest/immutable ID frontend, API, Nginx, dan MediaMTX dicatat.
- [x] Created/start time dan versi aktual keempat komponen dicatat.
- [x] Environment name, nama variable, port binding, topologi edge, dan ICE config dicatat tanpa nilai rahasia.
- [x] Matriks menetapkan status kecocokan setiap komponen dan menjelaskan keterbatasannya.
- [x] Dirty checkout serta revision-label gap dicatat sebagai risiko, bukan disamarkan sebagai `MATCH`.

Task 0.1 selesai. Artifact belum dinyatakan layak rilis; dokumen ini hanya menutup kebutuhan provenance dan membuka traceability gap yang harus ditangani pada release berikutnya.
