# Uji Manual - Progressive Web App (PWA)

> **Kontrak uji formal — MT-15**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan URL HTTPS yang dapat diakses perangkat, uji installability, offline shell, API fallback, cache v1/v2, dan Lighthouse di DevTools.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-15 |
| Capability owner | QA Web/Mobile + Frontend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT15-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Instalasi nyata berhasil, offline shell termuat, API offline memiliki fallback, cache lama terhapus, dan skor Lighthouse tercatat. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-15-S1 | YES | Instalasi nyata berhasil, offline shell termuat, API offline memiliki fallback, cache lama terhapus, dan skor Lighthouse tercatat. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT15-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 8.1 Progressive Web App (PWA)
> **Branch:** `feature/pwa`
> **Versi:** frontend `1.32.0`
> **Dokumen:** verifikasi installability, offline shell, web manifest, ikon aplikasi, dan service worker (cache-first shell, network-first API).

Scope implementasi: **installability + offline shell saja** (BUKAN full offline dengan IndexedDB). Data tetap membutuhkan koneksi jaringan/API.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `npm run dev` di `frontend/` → `http://localhost:3001`) |
| P2 | Koneksi aman | PWA/service worker hanya berjalan pada `https://` atau `http://localhost` (Secure Context) |
| P3 | Akun pengguna | Login sebagai admin atau teknisi |
| P4 | Browser Chrome/Edge desktop | Untuk uji installability & Lighthouse |
| P5 | Chrome/Edge Android | Untuk uji install di home screen HP (opsional tapi disarankan) |
| P6 | Alat Lighthouse | Chrome DevTools → tab Lighthouse, atau `npx lighthouse https://localhost` |
| P7 | Mode offline | Chrome DevTools → Network → pilih `Offline` (untuk uji offline shell) |

## 2. Pengujian Otomatis

```powershell
cd frontend
npm run lint
npm run build
```

Hasil yang diharapkan: `npm run lint` sukses (0 error) dan `npm run build` sukses (exit 0). Catatan: beberapa `warning` lint lama yang sudah ada tidak memengaruhi memenuhi kriteria/tidaknya.

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `npm run lint` | NOT RUN 0 error (77 warning lama di file lain) |
| `npm run build` | NOT RUN exit 0 |

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - Web Manifest tersedia dan benar

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost/manifest.json` | Respons JSON valid; `name="SIGAP"`, `short_name="SIGAP"`, `start_url="/dashboard"`, `display="standalone"` | NOT RUN |
| 2 | Cek nilai warna | `background_color="#162218"`, `theme_color="#467235"` | NOT RUN |
| 3 | Cek daftar ikon | Terdapat ikon `192x192` dan `512x512` (PNG), dengan `purpose: "any"` dan `"maskable"` | NOT RUN |
| 4 | Buka DevTools → Application → Manifest | Tidak ada error "Manifest not valid"; `start_url`, `display`, ikon terdeteksi | NOT RUN |

### S2 - Meta tags & link manifest di halaman

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost` dan lihat sumber halaman (View Source) | Ada `<link rel="manifest" href="/manifest.json">`, `<meta name="theme-color" content="#467235">`, dan `<meta name="viewport" ...>` | NOT RUN |
| 2 | Cek `apple-touch-icon` | Ada `<link rel="apple-touch-icon" ...>` menunjuk ikon PNG | NOT RUN |
| 3 | Cek title & description | `<title>` berisi "SIGAP"; meta description ada | NOT RUN |

### S3 - Service worker terdaftar

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost/sw.js` | File service worker tersaji (JavaScript), status 200 | NOT RUN |
| 2 | DevTools → Application → Service Workers | `sw.js` terdaftar dengan status "activated and is running", scope `/` | NOT RUN (verifikasi kode — tidak pakai browser) |
| 3 | Muat ulang halaman beberapa kali | Service worker tetap aktif; tidak ada error registrasi di Console | NOT RUN (verifikasi kode) |
| 4 | DevTools → Application → Cache Storage | Terdapat cache `sigap-shell-v1` berisi aset shell (halaman/ikon) | NOT RUN (cache aktif di produksi: `sigap-shell-v2`) |

### S4 - Installability (desktop)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Klik ikon "Install" pada address bar (atau menu ⋮ → Install SIGAP) | Dialog install muncul (memenuhi kriteria installable) | NOT RUN — Tidak dapat diuji (butuh browser desktop) |
| 2 | Klik Install | Aplikasi terbuka dalam jendela terpisah (`display: standalone`, tanpa address bar) | NOT RUN — Tidak dapat diuji (butuh browser desktop) |
| 3 | Buka ikon aplikasi dari desktop/menu | Aplikasi terbuka langsung ke `/dashboard` dengan ikon SIGAP | NOT RUN — Tidak dapat diuji (butuh browser desktop) |
| 4 | Cek judul jendela & ikon taskbar | Ikon SIGAP tampil; title "SIGAP" | NOT RUN — Tidak dapat diuji (butuh browser desktop) |

### S5 - Install di Android (opsional, sangat disarankan)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost` di Chrome Android | Muncul prompt/banner "Tambahkan ke layar utama" atau menu Install | NOT RUN — Tidak dapat diuji (butuh perangkat Android) |
| 2 | Pilih "Tambahkan ke layar utama" | Ikon SIGAP dengan latar Kanopi Malam (`#162218`) ditambahkan ke home screen | NOT RUN — Tidak dapat diuji (butuh perangkat Android) |
| 3 | Buka aplikasi dari home screen | Terbuka `standalone` (tanpa address bar), menuju `/dashboard` | NOT RUN — Tidak dapat diuji (butuh perangkat Android) |
| 4 | Periksa splash screen saat membuka aplikasi | Splash berwarna tema (`#467235`/`#162218`) dengan ikon SIGAP | NOT RUN — Tidak dapat diuji (butuh perangkat Android) |

### S6 - Offline shell (halaman dimuat dari cache)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka dan login SIGAP, lalu pastikan halaman shell termuat minimal sekali | Cache shell terisi | NOT RUN |
| 2 | DevTools → Network → pilih `Offline` | Status offline aktif | NOT RUN — Tidak dapat diuji (butuh browser) |
| 3 | Muat ulang halaman `/dashboard` (F5) | Halaman shell tetap termuat dari cache (layout, sidebar, CSS, JS) | NOT RUN (verifikasi kode — cache-first shell) |
| 4 | Muat ulang halaman `/login` | Halaman login shell termuat dari cache | NOT RUN (verifikasi kode — `/login` di daftar pre-cache) |
| 5 | Kembalikan Network ke `Online` | Aplikasi kembali memuat data dari jaringan | NOT RUN (verifikasi kode) |

### S7 - Strategi cache: API network-first

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Dalam kondisi online, buka halaman berdata (mis. Dashboard/Master CCTV) | Data API dimuat dari jaringan (segar), tidak dari cache | NOT RUN (verifikasi kode — API network-first) |
| 2 | Cek DevTools → Network untuk request `/api/*` | Request API tetap dikirim ke jaringan (network-first), bukan cache-only | NOT RUN (verifikasi kode) |
| 3 | Nonaktifkan jaringan (Offline) lalu akses halaman berdata | Jika ada data di cache, dimuat; bila tidak, muncul pesan/fallback (mis. 503 "offline") tanpa crash | NOT RUN (verifikasi kode — fallback JSON 503) |
| 4 | Kembalikan online, muat ulang | Data terbaru dimuat kembali dari jaringan | NOT RUN (verifikasi kode) |

### S8 - Update service worker & cache

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Ubah `CACHE_VERSION` pada `public/sw.js` menjadi `sigap-shell-v2`, lalu rebuild | Build sukses | NOT RUN (produksi sudah memakai `sigap-shell-v2`) |
| 2 | Muat ulang aplikasi | Service worker baru terpasang; cache lama `sigap-shell-v1` dihapus saat `activate` | NOT RUN (verifikasi kode — activate menghapus cache lama) |
| 3 | DevTools → Cache Storage | Hanya cache `sigap-shell-v2` yang tersisa | NOT RUN (verifikasi kode) |
| 4 | Kembalikan `CACHE_VERSION` ke `sigap-shell-v1` (jangan di-commit perubahan uji) | Aplikasi tetap berfungsi normal | NOT RUN — Tidak dapat diuji (butuh rebuild + browser) |

### S9 - Audit Lighthouse PWA

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost` di Chrome, DevTools → Lighthouse | Pilih kategori **Progressive Web App** (dan Performance/SEO opsional) | NOT RUN — Tidak dapat diuji (butuh browser + Lighthouse) |
| 2 | Generate audit | Skor PWA **≥ 80**; item PWA seperti "Web app manifest", "Service worker", "HTTPS", "Installable" terpenuhi | NOT RUN — Tidak dapat diuji (butuh browser + Lighthouse) |
| 3 | Catat skor | Skor PWA, Performance, dan Best Practices tercatat | NOT RUN — Tidak dapat diuji (butuh browser + Lighthouse) |

## 4. Kriteria Penerimaan

1. `public/manifest.json` memuat `name="SIGAP"`, `start_url="/dashboard"`, `display="standalone"`, `background_color="#162218"`, `theme_color="#467235"`, dan ikon `192x192` + `512x512`.
2. `layout.tsx` meng-export manifest, ikon (termasuk apple-touch-icon), dan `theme-color` di `viewport`; halaman merender `<link rel="manifest">`.
3. Service worker `public/sw.js` terdaftar di browser (scope `/`), berstatus aktif, dan menampilkan cache shell.
4. Strategi cache: shell (HTML/CSS/JS) cache-first, API network-first.
5. Aplikasi dapat di-install (desktop dan/atau Android) dan shell termuat saat offline.
6. `npm run lint` dan `npm run build` memenuhi kriteria.
7. Lighthouse PWA audit score **≥ 80**.

## Catatan

- Service worker hanya aktif pada Secure Context (`https://` atau `http://localhost`). Akses via `http://192.168.x.x` tidak mendukung PWA.
- Data aplikasi tetap membutuhkan jaringan (API network-first); mode offline hanya menyediakan shell, bukan data penuh (tanpa IndexedDB).
- Untuk melihat perubahan service worker saat pengembangan, centang "Update on reload" di DevTools → Application → Service Workers.
- Data cache shell dapat direset via DevTools → Application → Storage → "Clear site data", atau dengan mengganti `CACHE_VERSION` lalu rebuild.
