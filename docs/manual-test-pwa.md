# Uji Manual - Progressive Web App (PWA)

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

Hasil yang diharapkan: `npm run lint` sukses (0 error) dan `npm run build` sukses (exit 0). Catatan: beberapa `warning` lint lama yang sudah ada tidak memengaruhi lulus/tidaknya.

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Web Manifest tersedia dan benar

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost/manifest.json` | Respons JSON valid; `name="SIGAP"`, `short_name="SIGAP"`, `start_url="/dashboard"`, `display="standalone"` | |
| 2 | Cek nilai warna | `background_color="#162218"`, `theme_color="#467235"` | |
| 3 | Cek daftar ikon | Terdapat ikon `192x192` dan `512x512` (PNG), dengan `purpose: "any"` dan `"maskable"` | |
| 4 | Buka DevTools → Application → Manifest | Tidak ada error "Manifest not valid"; `start_url`, `display`, ikon terdeteksi | |

### S2 - Meta tags & link manifest di halaman

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost` dan lihat sumber halaman (View Source) | Ada `<link rel="manifest" href="/manifest.json">`, `<meta name="theme-color" content="#467235">`, dan `<meta name="viewport" ...>` | |
| 2 | Cek `apple-touch-icon` | Ada `<link rel="apple-touch-icon" ...>` menunjuk ikon PNG | |
| 3 | Cek title & description | `<title>` berisi "SIGAP"; meta description ada | |

### S3 - Service worker terdaftar

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost/sw.js` | File service worker tersaji (JavaScript), status 200 | |
| 2 | DevTools → Application → Service Workers | `sw.js` terdaftar dengan status "activated and is running", scope `/` | |
| 3 | Muat ulang halaman beberapa kali | Service worker tetap aktif; tidak ada error registrasi di Console | |
| 4 | DevTools → Application → Cache Storage | Terdapat cache `sigap-shell-v1` berisi aset shell (halaman/ikon) | |

### S4 - Installability (desktop)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik ikon "Install" pada address bar (atau menu ⋮ → Install SIGAP) | Dialog install muncul (memenuhi kriteria installable) | |
| 2 | Klik Install | Aplikasi terbuka dalam jendela terpisah (`display: standalone`, tanpa address bar) | |
| 3 | Buka ikon aplikasi dari desktop/menu | Aplikasi terbuka langsung ke `/dashboard` dengan ikon SIGAP | |
| 4 | Cek judul jendela & ikon taskbar | Ikon SIGAP tampil; title "SIGAP" | |

### S5 - Install di Android (opsional, sangat disarankan)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost` di Chrome Android | Muncul prompt/banner "Tambahkan ke layar utama" atau menu Install | |
| 2 | Pilih "Tambahkan ke layar utama" | Ikon SIGAP dengan latar Kanopi Malam (`#162218`) ditambahkan ke home screen | |
| 3 | Buka aplikasi dari home screen | Terbuka `standalone` (tanpa address bar), menuju `/dashboard` | |
| 4 | Periksa splash screen saat membuka aplikasi | Splash berwarna tema (`#467235`/`#162218`) dengan ikon SIGAP | |

### S6 - Offline shell (halaman dimuat dari cache)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka dan login SIGAP, lalu pastikan halaman shell termuat minimal sekali | Cache shell terisi | |
| 2 | DevTools → Network → pilih `Offline` | Status offline aktif | |
| 3 | Muat ulang halaman `/dashboard` (F5) | Halaman shell tetap termuat dari cache (layout, sidebar, CSS, JS) | |
| 4 | Muat ulang halaman `/login` | Halaman login shell termuat dari cache | |
| 5 | Kembalikan Network ke `Online` | Aplikasi kembali memuat data dari jaringan | |

### S7 - Strategi cache: API network-first

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Dalam kondisi online, buka halaman berdata (mis. Dashboard/Master CCTV) | Data API dimuat dari jaringan (segar), tidak dari cache | |
| 2 | Cek DevTools → Network untuk request `/api/*` | Request API tetap dikirim ke jaringan (network-first), bukan cache-only | |
| 3 | Nonaktifkan jaringan (Offline) lalu akses halaman berdata | Jika ada data di cache, dimuat; bila tidak, muncul pesan/fallback (mis. 503 "offline") tanpa crash | |
| 4 | Kembalikan online, muat ulang | Data terbaru dimuat kembali dari jaringan | |

### S8 - Update service worker & cache

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Ubah `CACHE_VERSION` pada `public/sw.js` menjadi `sigap-shell-v2`, lalu rebuild | Build sukses | |
| 2 | Muat ulang aplikasi | Service worker baru terpasang; cache lama `sigap-shell-v1` dihapus saat `activate` | |
| 3 | DevTools → Cache Storage | Hanya cache `sigap-shell-v2` yang tersisa | |
| 4 | Kembalikan `CACHE_VERSION` ke `sigap-shell-v1` (jangan di-commit perubahan uji) | Aplikasi tetap berfungsi normal | |

### S9 - Audit Lighthouse PWA

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `https://localhost` di Chrome, DevTools → Lighthouse | Pilih kategori **Progressive Web App** (dan Performance/SEO opsional) | |
| 2 | Generate audit | Skor PWA **≥ 80**; item PWA seperti "Web app manifest", "Service worker", "HTTPS", "Installable" terpenuhi | |
| 3 | Catat skor | Skor PWA, Performance, dan Best Practices tercatat | |

## 4. Kriteria Lulus

1. `public/manifest.json` memuat `name="SIGAP"`, `start_url="/dashboard"`, `display="standalone"`, `background_color="#162218"`, `theme_color="#467235"`, dan ikon `192x192` + `512x512`.
2. `layout.tsx` meng-export manifest, ikon (termasuk apple-touch-icon), dan `theme-color` di `viewport`; halaman merender `<link rel="manifest">`.
3. Service worker `public/sw.js` terdaftar di browser (scope `/`), berstatus aktif, dan menampilkan cache shell.
4. Strategi cache: shell (HTML/CSS/JS) cache-first, API network-first.
5. Aplikasi dapat di-install (desktop dan/atau Android) dan shell termuat saat offline.
6. `npm run lint` dan `npm run build` lulus.
7. Lighthouse PWA audit score **≥ 80**.

## Catatan

- Service worker hanya aktif pada Secure Context (`https://` atau `http://localhost`). Akses via `http://192.168.x.x` tidak mendukung PWA.
- Data aplikasi tetap membutuhkan jaringan (API network-first); mode offline hanya menyediakan shell, bukan data penuh (tanpa IndexedDB).
- Untuk melihat perubahan service worker saat pengembangan, centang "Update on reload" di DevTools → Application → Service Workers.
- Data cache shell dapat direset via DevTools → Application → Storage → "Clear site data", atau dengan mengganti `CACHE_VERSION` lalu rebuild.
