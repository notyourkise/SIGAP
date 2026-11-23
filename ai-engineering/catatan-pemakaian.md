# Catatan Pemakaian Token & Efisiensi Biaya — SIGAP AI Assistant

- **Nama / NIM:** Muhammad Fikri Haikal Ariadma / `10231063` (Sistem Informasi — Kelas Konversi)
- **Tanggal Pembaruan:** 23 November 2026
- **Konfigurasi Model Gateway:** Google AI Studio OpenAI-Compatible (`AI_CHAT_MODEL=gemini-3.1-flash-lite`, `AI_EMBEDDING_MODEL=gemini-embedding-001`, tier gratis / Free Tier).

---

## 1. Catatan Pemakaian Nyata (`cmd/ai-evaluate`)
| Sesi Pengujian | Mode | Jumlah Kasus | Lulus | Cache Hit Dokumen | Chat Prompt Token | Chat Completion Token | Total Token Chat | Rata-rata Latensi | Biaya Langsung |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Uji 1 (`eval_penuh_clean`) | `penuh` | 17 | 16/17 *(Q17 gagal arg tool)* | 120 *(Q02–Q16)* | 37.053 *(16 terukur)* | 3.422 *(16 terukur)* | 40.475 | 6.169 ms | Rp 0 *(Free Tier)* |
| Uji 2 (`eval_ringkas_clean`) | `ringkas` | 17 | 15/17 *(Q10 503, Q17 arg tool)* | 112 *(14 cached)* | 40.312 *(15 terukur)* | 3.031 *(15 terukur)* | 43.343 | 5.877 ms | Rp 0 *(Free Tier)* |
| Uji 3 (`eval_qguard`, pasca-fix Q17) | `penuh` | 4 *(Q10,Q15–Q17)* | 4/4 | 24 *(Q15–Q17)* | 8.121 | 413 | 8.534 | 3.822 ms | Rp 0 *(Free Tier)* |

> **Catatan teknis `/embeddings`:** Endpoint OpenAI-compatible Google AI Studio mengembalikan nilai `usage.prompt_tokens = 0` untuk `/embeddings`, sehingga metrik token di atas adalah token aktual untuk `/chat/completions`, sementara efisiensi `/embeddings` diukur dari **`embedding_cache_hits`** (`8` dokumen korpus = `9.885` karakter di-skip per kueri).

---

## 2. Tiga Angka Biaya & Proyeksi Bulanan (Berdasarkan Data Nyata)
1. **Permintaan Khas (Rata-rata 16 kueri lolos mode `penuh`):**
   - `2.529` total token chat (`~2.316` token input + `~214` token output), dengan **0 pengiriman ulang** korpus statis (`9.885` karakter) ke `/embeddings` setelah cache hangat.
   - Minimum tercatat: `1.614` token (`Q06`, saat model langsung memilih tool spesifik `lihat_batas_monitoring` yang lebih ringkas).
2. **Permintaan Terburuk (Worst Case Terukur):**
   - Pada mode penuh multi-sumber `Q08`: `2.959` token (`2.342` input + `617` output).
   - Pada mode ringkas `Q09` (ketika agen memanggil **2 tool berurutan** `cari_dokumen` + `lihat_batas_monitoring` dalam 3 langkah): **`5.747` token** (`5.689` input + `58` output) — membuktikan mengapa batas `MaxSteps = 4` penting agar biaya tidak membengkak.
3. **Proyeksi `100 pengguna × 20 permintaan/bulan` (`2.000` permintaan/bulan):**
   - **Skenario khas (`2.529` token):** `~5.058.000` token chat/bulan (`~4.632.000` input + `~428.000` output).
   - **Skenario terburuk (`5.747` token):** `~11.494.000` token chat/bulan.
   - **Status biaya aktual:** Saat ini **Rp 0** karena memakai kuota Free Tier Google AI Studio (dengan konsekuensi batas RPM ~15 permintaan/menit); jika kelak beralih ke tier berbayar (*pay-as-you-go*), biaya rupiah tinggal dikalikan terhadap tarif per 1 juta token yang berlaku.

---

## 3. Teknik Penghematan yang Sudah Diterapkan & Buktinya
| Penghematan | Mekanisme di Kode | Dampak Terukur | Kualitas Turun? |
|---|---|---|---|
| **Shortcut Sapaan (`0` Token)** | `greetingResponse()` di `aiassistant/assistant.go` | Sapaan (`hai`, `halo`, `terima kasih`) memakai **0 token** chat maupun embedding (`TestAnswerHandlesGreetingsWithoutGateway` PASS). | Tidak — pertanyaan substantif tetap masuk ke pipeline RAG penuh. |
| **Caching Embedding Korpus (`docEmbeddingCache`)** | Hash `SHA-256(ID + Text)` pada `corpus.json` (`9.885` karakter) | Setelah kueri ke-1, `8` dokumen (`9.885` karakter) tidak pernah dikirim ulang ke `/embeddings` (**120 cache hit** selama 1 run 17 kasus = penghematan >99% karakter embedding statis). | Tidak — skor retrieval cosine identik deterministik (`Q01–Q16` 100% lulus sumber). |
| **Mode Output `ringkas`** | Toggle UI + instruksi ringkas 2–3 kalimat pada `AnswerOptions.Mode` | Pada 15 kasus lolos yang sama, output completion turun dari `3.363` (`penuh`) menjadi `3.031` token (`ringkas`), dan pada kueri panjang `Q08` turun dari `617` ke `340` token (**-45%**). | Tidak — seluruh 15 kasus yang terhindar dari 503 tetap lulus verifikasi `Validate()`. |
