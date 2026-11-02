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

## Progres AI Engineering sampai Week 9

Halaman `/ai-assistant` menyediakan antarmuka tanya-jawab untuk pengguna SIGAP yang sudah login. Backend menggunakan gateway yang kompatibel dengan `embeddings` dan `chat/completions`, menghitung kemiripan embedding, memilih dokumen kebutuhan, menawarkan dua tool baca-saja kepada model, dan meminta jawaban JSON terstruktur. Kutipan diperiksa terhadap teks sumber; jika tidak valid, sistem menyatakan jawaban tidak dapat ditentukan.

Sumber awal adalah lima potongan **kebutuhan rancangan proyek**, bukan SOP resmi atau data CCTV terkini. Gunakan hanya pertanyaan yang boleh dikirim ke penyedia gateway. Konfigurasi `AI_GATEWAY_URL`, `AI_GATEWAY_KEY`, `AI_CHAT_MODEL`, dan `AI_EMBEDDING_MODEL` di server; jangan commit kunci. Tanpa konfigurasi tersebut sistem memberikan pesan layanan tidak tersedia.

Tes menggunakan gateway HTTP tiruan. Koneksi ke model nyata, kualitas retrieval pada korpus lengkap, serta validasi dosen belum diuji. Modul 1–9 juga memuat tugas dan laporan proses individu (termasuk lembar tema, eksperimen before/after, evaluasi dan presentasi UTS) yang belum otomatis selesai hanya dengan kode ini; cakupan personal bergantung pada kode peserta K dan dokumen masing-masing.

## Batas progres dan langkah selanjutnya

- Gateway, retrieval embedding, dua tool baca-saja, dan validasi kutipan sudah dibuat sebagai prototipe. Korpus SOP sebenarnya, evaluasi menyeluruh, penggunaan tool operasional secara langsung, dan agent multi-langkah masih perlu dilengkapi.
- Pengelompokan insiden multi-kamera, pengingat penugasan, pengukuran GSM, dan pengaturan monitoring terpisah masih perlu implementasi.
- Target notifikasi maksimal lima menit merupakan kebutuhan yang belum dibuktikan pada perangkat dan jaringan nyata.
- Fitur insiden pada aplikasi disiapkan melalui feature flag dan belum diaktifkan pada lingkungan operasional.
