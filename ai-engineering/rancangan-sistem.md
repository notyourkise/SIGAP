# Rancangan Sistem SIGAP — AI Engineering (Lapisan 1–5)

- **Penulis:** Muhammad Fikri Haikal Ariadma (NIM `10231063` — Sistem Informasi, Kelas Konversi)
- **Pembaruan:** 23 November 2026 (diselaraskan dengan persetujuan dosen atas pemisahan arsitektur dan batas maksimum `8` dokumen, `3` tool, `17` kasus uji, mode `penuh` + `ringkas`).

---

## 1. Pemisahan Tegas: Deterministik vs AI
1. **Lapisan Deterministik (Non-AI — `backend/incidents/`, `backend/handlers/stream_verify.go`):**
   - Verifikasi RTSP/MediaMTX berkala, aturan pemulihan 5 menit (`>= 6` sampel `live` beruntun dengan celah `<= 105 detik`), korelasi NVR bersama, antrean `telegram_outbox`, serta perubahan status tiket (`open` → `in_progress` → `pending_verification` → `resolved`) dijalankan murni oleh kode aturan dan transaksi PostgreSQL agar bebas halusinasi.
2. **Lapisan AI Generatif Baca-Saja (`backend/aiassistant/`, `backend/handlers/ai_assistant*.go`, `/ai-assistant`):**
   - Membantu admin/teknisi menjawab pertanyaan dokumentasi SIGAP dan merangkum kondisi gangguan CCTV 7 hari terakhir beserta penyebab dominan menggunakan bahasa alami dan sitasi persis.

## 2. Alur Eksekusi & Orkestrasi Agentic
1. Pengguna terautentikasi (`JWT` + `SiteContext`) mengirim pertanyaan beserta pilihan mode (`penuh` atau `ringkas`) ke `POST /api/ai-assistant/ask`.
2. **Shortcut Sapaan (0 Token):** Sapaan ringan (`hai`, `halo`, `terima kasih`, dll.) dijawab langsung oleh `greetingResponse` tanpa memanggil gateway model.
3. **Embedding + Cache (`docEmbeddingCache`):** Hanya teks pertanyaan dan dokumen yang belum ter-cache (kunci `BaseURL + EmbeddingModel + SHA-256(ID + Text)`) yang dikirim ke `/embeddings`.
4. **Retrieval + Injeksi Konteks Live:** Skor cosine memilih 4 potongan teratas; khusus pertanyaan operasional/7 hari (`isOperationalSummaryQuestion`), ringkasan teragregasi `live-db://ringkasan-7-hari` dari PostgreSQL ikut disertakan ke 4 dokumen terpilih.
5. **Siklus Reason–Act Terbatas (`AllowMultiStepToolLoop`, `MaxSteps = 4`):**
   - Langkah 1 memanggil tool wajib (`tool_choice: "required"`).
   - Langkah 2..4 memakai `tool_choice: "auto"` hingga agen memutuskan selesai atau mencapai batas `MaxSteps = 4` (`max_steps_reached: true`).
6. **Tool Baca-Saja (3 Tool dengan Parameter Kosong `{}`):**
   - `cari_dokumen`: mengambil maksimal 4 potongan terpilih dan mencatatnya sebagai `exposed`.
   - `lihat_batas_monitoring`: menyaring hanya potongan terkait monitoring/verifikasi dan mencatatnya ke `exposed`.
   - `daftar_sumber`: menampilkan daftar ID sumber saja tanpa memasukkannya ke `exposed` (sehingga tidak sah dijadikan bukti kutipan).
7. **Kendali Output, Guardrails & Human-in-the-Loop:**
   - Langkah akhir menyusun JSON ketat `{"jawaban", "dapat_ditentukan", "bukti"}` dan mencatat `usage` (`embedding_prompt_tokens`, `embedding_cache_hits`, `chat_prompt_tokens`, `chat_completion_tokens`, `total_tokens`, `steps_used`, `tools_called`, `guardrail_blocked`, `latency_ms`).
   - `Validate()` menolak setiap `bukti` yang tidak memuat `kutipan` substring persis minimal 8 karakter dari dokumen yang benar-benar dikembalikan tool pada sesi tersebut.
   - Argumen tool yang tidak sah atau upaya menembus allowlist ditahan dengan `guardrail_blocked: true` serta jawaban penolakan aman (`dapat_ditentukan: false`).
   - Seluruh aksi operasional (penugasan teknisi, eskalasi ganti unit, penutupan tiket) tetap berada di halaman `/incidents` dan wajib melalui tombol verifikasi Admin manusia.
