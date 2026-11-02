# Progres Kapita Selekta AI Engineering: SIGAP

Pembaruan: 2 November 2026.

## Tujuan

Mengembangkan fitur di SIGAP untuk membantu operasional CCTV: mendeteksi gangguan, mencatat insiden, memberi informasi kepada teknisi melalui grup Telegram, dan membantu admin mengelola penanganan. Lapisan agent AI direncanakan untuk membantu pemahaman SOP dan analisis gangguan.

## Progres yang tersedia

- Engine status insiden dan aturan pemulihan: `backend/incidents/`.
- Insiden persisten, keanggotaan kamera unik, dan antrean pengiriman: `backend/migrations/000031_cctv_incidents.up.sql`.
- API admin untuk melihat, menugaskan, dan memverifikasi penutupan insiden: `backend/handlers/incidents.go` dan `backend/routes/incidents.go`.
- Integrasi dengan verifier CCTV yang sudah ada: `backend/handlers/stream_verify.go`.
- Halaman admin Tiket Gangguan: `frontend/src/app/incidents/page.tsx`.
- Demo interaktif lokal: `backend/cmd/incident-demo/`.

Demo memakai data kamera tiruan, jam simulasi, dan penerima HTTP lokal untuk menirukan Telegram. Demo menggunakan reducer pemulihan dan pengirim HTTP yang sama dengan kode implementasi; demo tidak menghubungi kamera atau Telegram nyata.

## Menjalankan demo

Persyaratan: Go sesuai `backend/go.mod`.

```bash
cd backend
go run ./cmd/incident-demo
```

Buka `http://127.0.0.1:8095`, kemudian tekan **Simulasikan offline**, **Tugaskan Teknisi Demo**, **Bukti online**, **+2 menit & periksa** dua kali, **+1 menit & periksa**, lalu **Verifikasi & tutup**. Tombol pemantau gagal dapat digunakan untuk melihat penanganan bukti yang tidak tersedia.

## Verifikasi

`go test ./...` dan `go vet ./...` lulus pada workspace pengembangan. Tes juga dijalankan dengan PostgreSQL khusus yang terisolasi, termasuk pemeriksaan pembuatan insiden secara paralel, pemulihan, penolakan penutupan lintas site, dan antrean pengiriman dengan server HTTP tiruan. Pengujian TypeScript dan ESLint pada halaman baru lulus. Pengujian kamera/NVR dan grup Telegram nyata belum dilakukan.

## Batas progres dan langkah selanjutnya

- Belum ada integrasi LLM, RAG/SOP, tool calling, atau evaluasi agent AI; ini adalah fondasi operasional yang akan digunakan oleh agent.
- Pengelompokan insiden multi-kamera, pengingat penugasan, pengukuran GSM, dan pengaturan monitoring terpisah masih perlu implementasi.
- Target notifikasi maksimal lima menit merupakan kebutuhan yang belum dibuktikan pada perangkat dan jaringan nyata.
- Fitur insiden pada aplikasi disiapkan melalui feature flag dan belum diaktifkan pada lingkungan operasional.
