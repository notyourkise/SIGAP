# LEMBAR TEMA PROYEK — SIGAP

- **Nama:** Muhammad Fikri Haikal Ariadma
- **NIM:** 10231063
- **Program Studi:** Sistem Informasi
- **Jalur / Kode Peserta `K`:** Mahasiswa Kelas Konversi (tidak memiliki kode peserta `K` reguler; berdasarkan konfirmasi dan persetujuan dosen pengampu pada 23 Nov 2026, seluruh batas minimum diturunkan menggunakan **batas atas maksimum** dari rumus Lampiran B).

## Angka Lingkup Wajib (Batas Maksimum Semua Kemungkinan `K`)
- **Dokumen rujukan minimum:** `5 + max(K mod 4) = 5 + 3 = 8` dokumen → **Terpenuhi (`8` snapshot dokumen di `backend/aiassistant/corpus.json` + 1 sumber operasional live `live-db://ringkasan-7-hari`).**
- **Kasus uji minimum:** `12 + max(K mod 6) = 12 + 5 = 17` kasus → **Terpenuhi (`17` kasus di `ai-engineering/evaluasi/pertanyaan.json`).**
- **Tool minimum:** `2 + max(K mod 2) = 2 + 1 = 3` tool → **Terpenuhi (`cari_dokumen`, `lihat_batas_monitoring`, `daftar_sumber`).**
- **Mode output:** Mendukung **mode penuh (`penuh`)** dan **mode ringkas (`ringkas`)** pada backend (`AnswerOptions.Mode`) maupun toggle UI `/ai-assistant`.

---

## 1. Persoalan
- **Judul kerja produk:** Asisten Operasional SIGAP — Penelusur Dokumentasi Sistem & Ringkasan Gangguan CCTV Terverifikasi.
- **Siapa yang mengalami persoalan ini:** Admin operasional dan teknisi pemeliharaan CCTV SIGAP pada pemantauan kawasan multi-site.
- **Bagaimana persoalan ini diselesaikan sekarang:** Petugas harus membuka beberapa file dokumentasi teknis (`ACCESS.md`, `BACKUP.md`, semantik log, batasan codec, laporan verifikasi aktif) serta menelusuri riwayat log stream dan tiket insiden satu per satu untuk memahami prosedur atau merangkum gangguan 7 hari terakhir.
- **Mengapa cara sekarang tidak memadai:** Dokumentasi tersebar di banyak berkas teknis sementara ringkasan penyebab gangguan tersebar di ribuan baris `cctv_stream_logs` dan `cctv_incidents`, sehingga memperlambat koordinasi saat insiden terjadi.

## 2. Mengapa AI Generatif (dan Pemisahan Batas Deterministik)
- **Yang tetap deterministik (BUKAN AI):** deteksi kamera `offline`/pulih 5 menit (`incidents/state.go`), penjadwalan antrean notifikasi grup Telegram (`incidents/telegram.go`), serta perubahan status tiket dan penugasan teknisi (`incidents/actions.go`) — karena keputusan ini menuntut kepastian matematis dan tidak boleh berhalusinasi.
- **Yang menggunakan AI generatif:** menyusun jawaban bahasa alami beserta rujukan kutipan persis dari dokumentasi proyek dan ringkasan log gangguan 7 hari terakhir melalui asisten baca-saja (`backend/aiassistant`), karena pertanyaan petugas datang dalam variasi kalimat alami dan merangkum lintas dokumen/log.

## 3. Ukuran Keberhasilan
- **Bukti produk berhasil:** setiap jawaban faktual wajib menyertakan kutipan persis (`sumber` + `kutipan` minimal 8 karakter) yang lolos verifikasi server `Validate()`; pertanyaan di luar sumber atau serangan injeksi ditolak tegas dengan `dapat_ditentukan: false`; tidak satu pun tool mengubah state database/tiket.
- **Siapa yang memeriksa benar/salah:** Admin/pemilik sistem SIGAP dan penguji independen melalui set uji 17 kasus beserta kutipan sumber aslinya.
- **Risiko jika salah dan penanggungnya:** jika petugas langsung mempercayai jawaban keliru tanpa mengecek sumber, langkah penanganan bisa tidak sesuai SOP; karena itu UI menampilkan penafian bahwa AI dapat keliru dan seluruh aksi operasional (`Assign`, `Resolve`, `Escalate`) tetap memerlukan verifikasi dan otorisasi Admin manusia (*human-in-the-loop*).

## 4. Dokumen Rujukan (`backend/aiassistant/corpus.json`)
| # | ID Sumber | Dokumen Asal | Tahun / Status |
|---|---|---|---|
| 1 | `docs/01-documentation/ACCESS.md#inicio` | Panduan Akses Web SIGAP | 2026 · Dokumentasi proyek |
| 2 | `docs/01-documentation/BACKUP.md#inicio` | BACKUP — Backup Otomatis & Restore | 2026 · Dokumentasi proyek |
| 3 | `docs/01-documentation/cctv-stream-log-semantics.md#inicio` | Semantik Log Event Stream CCTV | 2026 · Dokumentasi proyek |
| 4 | `docs/01-documentation/cctv-codec-snapshot-limitations.md#inicio` | Batasan Codec Streaming dan Snapshot CCTV | 2026 · Dokumentasi proyek |
| 5 | `docs/05-reports/CCTV-ACTIVE-VERIFICATION-2026-10-22.md#inicio` | Verifikasi Aktif Status Streaming CCTV | 2026 · Laporan implementasi |
| 6 | `docs/04-manual-tests/manual-test-cctv-active-verification.md#inicio` | Manual Test Verifikasi Aktif CCTV | 2026 · Dokumen uji |
| 7 | `docs/05-reports/FIX-BB-03-BB-04-MULTI-SITE-ISOLATION.md#inicio` | Fix Multi-Site Isolation Issues | 2026 · Laporan perbaikan |
| 8 | `PROGRES-AI-ENGINEERING.md#inicio` | Progres Kapita Selekta AI Engineering: SIGAP | 2026 · Catatan arsitektur |
| + | `live-db://ringkasan-7-hari` *(konteks operasional dinamis)* | Ringkasan teragregasi `cctv_incidents` & `cctv_stream_logs` 7 hari terakhir (tanpa kredensial/RTSP/password) | Live per kueri |

## 5. Uji Lima Kriteria Penolakan
1. **AI tidak dipaksakan pada persoalan deterministik:** LOLOS — deteksi kamera offline/pulih, antrean Telegram, dan perubahan status tiket tetap dijalankan 100% oleh engine aturan Go dan PostgreSQL.
2. **Output dapat dibuktikan benar atau salah:** LOLOS — setiap jawaban JSON diperiksa secara skematis oleh `Validate()` terhadap kutipan persis dari dokumen sumber dan dievaluasi pada 17 kasus uji.
3. **Sumber pengetahuan tersedia dan sah dipakai:** LOLOS — 8 dokumen berasal dari dokumentasi proyek SIGAP milik sendiri dan data operasional 7 hari disaring tanpa kredensial.
4. **Tidak bergantung pada data pribadi orang sungguhan:** LOLOS — tidak mengirim data pribadi sensitif maupun kredensial RTSP/NVR/bot ke model gateway.
5. **Lingkup terukur:** LOLOS — dibatasi hanya sebagai asisten tanya-jawab baca-saja untuk admin/teknisi SIGAP pada `/ai-assistant`.
