# Rancangan RAG SIGAP

- **Pembaruan:** 23 November 2026
- **Implementasi:** `backend/aiassistant/assistant.go`, `backend/aiassistant/corpus.go`, `backend/handlers/ai_assistant.go`, `backend/handlers/ai_assistant_summary.go`.

---

## 1. Masalah dan Batas
Admin dan teknisi membutuhkan pencarian cepat pada dokumentasi operasional SIGAP serta ringkasan insiden 7 hari terakhir tanpa harus membuka banyak file log atau dokumentasi secara manual. Asisten AI bertindak murni sebagai pembantu baca-saja dengan kutipan sumber terverifikasi dan tidak pernah menggantikan otorisasi manusia.

## 2. Sumber Pengetahuan dan Kelayakan
- **Korpus Statis (`backend/aiassistant/corpus.json`):** Berisi **8 snapshot dokumen** berbeda yang diverifikasi kesesuaian teksnya oleh unit test otomatis `TestCorpusHasEightDistinctVerifiableSourceSnapshots`.
- **Dokumen Konteks Operasional Dinamis (`live-db://ringkasan-7-hari`):** Dihasilkan oleh `buildSevenDaySummaryDocument()` di `backend/handlers/ai_assistant_summary.go` dari tabel `cctv_incidents`, `cctv_incident_cameras`, `cctv_incident_events`, dan `cctv_stream_logs` selama 7 hari terakhir (di-filter berdasarkan `site_id` pengguna). Dokumen ini hanya berisi ringkasan agregat dan kode/nama kamera, **tanpa** `rtsp_url`, `rtsp_password`, kredensial NVR, maupun data pribadi.

## 3. Keputusan Rancangan dan Alasan
1. **Pemotongan (*Chunking*):** Setiap dokumen statis disimpan sebagai satu potongan awal berukuran ~1.187–1.360 karakter (~9.885 karakter total untuk 8 dokumen). Sederhana dan mudah diaudit, dengan catatan bagian bawah dokumen asli yang melebihi panjang potongan belum tercakup.
2. **Embedding & Caching (`docEmbeddingCache`):** Memakai model `gemini-embedding-001` via endpoint kompatibel OpenAI. Vektor embedding tiap dokumen di-cache di memori dengan kunci `SHA-256(BaseURL + EmbeddingModel + DocID + DocText)` (maksimal 1.024 entri). Setelah panggilan pertama, 8 dokumen statis tidak pernah dikirim ulang ke `/embeddings` (**8 cache hit per kueri**).
3. **Retrieval:** Pencarian cosine similarity mengambil maksimal 4 potongan teratas (`top-4`). Untuk kueri yang terdeteksi menanyakan kondisi operasional/7 hari (`isOperationalSummaryQuestion`), dokumen `live-db://ringkasan-7-hari` dijamin tetap masuk ke slot `selected` agar ringkasan insiden terkini tidak tersisih oleh dokumen statis.
4. **Konteks & Anti-Injeksi:** Data dokumen dikembalikan lewat tool baca-saja (`cari_dokumen`, `lihat_batas_monitoring`) dan diinstruksikan sebagai data tidak tepercaya agar instruksi palsu di dalam dokumen diabaikan.
5. **Output & Validasi Sitasi (`Validate()`):** Model wajib mengembalikan JSON `jawaban`, `dapat_ditentukan`, dan `bukti`. Server memverifikasi bahwa tiap `bukti.sumber` cocok dengan ID dokumen yang benar-benar terekspose lewat tool dan `bukti.kutipan` merupakan substring persis minimal 8 karakter.
6. **Mode Output & Penolakan:** Mendukung mode `penuh` dan `ringkas` (maksimal 2–3 kalimat padat). Pertanyaan tanpa dasar di dokumen atau argumen tool tak sah menghasilkan `dapat_ditentukan: false` dengan `bukti: []`.

## 4. Evaluasi Nyata & Penanganan Tantangan
- Pengujian dilakukan melalui `backend/cmd/ai-evaluate` terhadap **17 kasus** pada `ai-engineering/evaluasi/pertanyaan.json` menggunakan `gemini-3.1-flash-lite` + `gemini-embedding-001`.
- Penambahan jeda `-delay 7s` dan *backoff retry* di `ai-evaluate` mengatasi batas rate-limit tier gratis Google AI Studio (`HTTP 429`/`503`). Detail angka hasil lengkap terdokumentasi di `ai-engineering/laporan-evaluasi.md`.
