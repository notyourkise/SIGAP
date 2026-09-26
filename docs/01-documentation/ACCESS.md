# Panduan Akses Web SIGAP

## Akses Resmi

Untuk deployment Docker, aplikasi SIGAP hanya diakses melalui:

```text
https://localhost
```

`http://localhost` hanya berfungsi sebagai pintu redirect HTTP ke HTTPS. Akses langsung
ke `http://localhost:3000` atau `http://localhost:3001` bukan entrypoint aplikasi dan tidak
didukung untuk penggunaan operasional.

Port backend dan frontend hanya tersedia di dalam network Docker. Nginx adalah satu-satunya
service yang mem-publish port ke host:

- `80`: menerima HTTP dan mengarahkan request ke HTTPS.
- `443`: terminasi TLS dan entrypoint web resmi.

## Alur Reverse Proxy

```text
Browser
  |
  | https://localhost
  v
Nginx :443
  |-- /          -> frontend:3000
  |-- /api/*     -> api:3000
  |-- /swagger/* -> api:3000 (non-production)
  `-- /api/ws    -> api:3000 melalui WebSocket upgrade
```

Backend dan frontend tidak boleh diakses langsung dari host. Port `3000` pada kedua service
adalah port internal container, sedangkan port `3001` frontend tidak dipublish.

## Alasan Single-Origin

Frontend dan API menggunakan host, skema, dan port publik yang sama:

- Frontend: `https://localhost/`
- API: `https://localhost/api/`
- WebSocket: `wss://localhost/api/ws`

Single-origin memastikan browser mengirim cookie sesi httpOnly `sigap_session` secara konsisten.
Cookie CSRF `sigap_csrf` tetap dibaca oleh JavaScript untuk pola double-submit dan dikirim ulang
melalui header `X-CSRF-Token`. Token JWT tidak disimpan di `localStorage` atau `sessionStorage`.

Pada Docker, pastikan `backend/.env.docker` menggunakan
`CORS_ALLOWED_ORIGINS=https://localhost`. Template resminya sudah menggunakan origin tersebut.

Cookie sesi dan CSRF memiliki `MaxAge` 18000 detik (5 jam), sama dengan masa berlaku JWT.

## Kamera dan Secure Context

Gunakan `https://localhost` saat memakai scanner QR berbasis kamera. Browser mengizinkan
`getUserMedia()` pada HTTPS, sedangkan akses melalui IP LAN seperti `http://192.168.x.x:3001`
dianggap insecure context dan dapat ditolak.

## Swagger

Pada environment non-production, Swagger UI tersedia melalui Nginx:

```text
https://localhost/swagger/index.html
```

Route Swagger tetap dinonaktifkan ketika `APP_ENV=production`.

## Verifikasi Deployment

Jalankan dari root repository:

```bash
docker compose up -d --build
docker compose ps
```

Kemudian buka `https://localhost` dan uji alur login, refresh halaman, logout, login ulang,
serta koneksi WebSocket. Browser mungkin perlu mempercayai sertifikat development dari
`nginx/certs/` terlebih dahulu.

## Development Tanpa Docker

Backend `:3000` dan frontend `:3001` masih dapat digunakan untuk debugging lokal terisolasi.
Mode tersebut bukan jalur deployment standar dan tidak menggantikan verifikasi wajib melalui
`https://localhost`, khususnya untuk pengujian cookie httpOnly, CSRF, TLS, dan WebSocket.
