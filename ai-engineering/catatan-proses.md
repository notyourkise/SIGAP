# Catatan Proses — Implementasi & Penyempurnaan SIGAP AI Engineering

- **Nama / NIM / Jalur:** Muhammad Fikri Haikal Ariadma / `10231063` / Sistem Informasi — **Kelas Konversi**
- **Tanggal Pembaruan:** 23 November 2026
- **Acuan Persetujuan Dosen:** Batas maksimum semua kemungkinan `K` (`8` dokumen, `3` tool, `17` kasus uji, mode output `penuh` & `ringkas`) serta pemisahan tegas antara engine insiden deterministik dan asisten AI baca-saja.

---

## 1. Ringkasan Pekerjaan & Evolusi Produk
- **2 Nov 2026 (Fondasi & RAG Awal):** Pembuatan paket `backend/aiassistant` dengan korpus 8 snapshot (`corpus.json`), prompt `v1.txt` → `v2.txt`, 3 tool baca-saja, validasi kutipan `Validate()`, dan set uji awal 10 kasus. Saat itu pengujian baru memakai server HTTP tiruan (`httptest`).
- **13–15 Nov 2026 (Integrasi Model Nyata, UX Chat & Efisiensi):** Menghubungkan gateway nyata Google AI Studio (`gemini-3.1-flash-lite` + `gemini-embedding-001`), memperbaiki passthrough `extra_content.google.thought_signature` pada tool-call Gemini, merancang ulang UI `/ai-assistant` menjadi gelembung chat dengan Question Library, menambahkan shortcut sapaan `0` token, menambahkan cache embedding dokumen (`docEmbeddingCache`), serta menyuntikkan ringkasan operasional 7 hari live dari PostgreSQL (`live-db://ringkasan-7-hari`).
- **23 Nov 2026 (Penyelesaian Blok D & E untuk Konversi):** Menambahkan siklus tool multi-langkah berbatas (`MaxSteps = 4`, `AllowMultiStepToolLoop`), opsi mode `penuh` / `ringkas` di backend dan UI beserta *disclaimer* AI, pelacakan pemakaian token `UsageStats`, perluasan set uji menjadi **17 kasus**, evaluasi langsung ke Gemini, serta perbaikan guardrail penolakan aman untuk argumen tool tidak sah.

## 2. Tahap BREAK & FIX (Temuan Nyata Saat Pengujian ke Gemini)
| # | Percobaan / Pengujian | Prediksi Awal | Hasil Nyata yang Terjadi | Akar Masalah & Perbaikan (`FIX`) |
|---|---|---|---|---|
| 1 | Pemanggilan `tool_calls` pada `gemini-3.1-flash-lite` | Cukup kirim ulang `role`, `id`, `function` seperti OpenAI standar | Gateway menolak follow-up tool call karena kehilangan field `extra_content` (`thought_signature`) | Mempertahankan `ExtraContent json.RawMessage` pada struct tool-call di `assistant.go` |
| 2 | Menjalankan 17 kasus uji `cmd/ai-evaluate` tanpa jeda | Seluruh 17 kasus berjalan berurutan lancar | Kasus `Q08–Q13` gagal `HTTP 429` / `503` karena menembus batas RPM tier gratis AI Studio | Menambahkan `-delay 7s` dan *backoff retry* 15 detik untuk error `429`/`503` di `cmd/ai-evaluate/main.go` |
| 3 | Menjalankan kasus serangan `Q17` (*extract system prompt & credentials*) | Model langsung mengembalikan JSON `dapat_ditentukan: false` | Pada dua kali evaluasi (`penuh` dan `ringkas`), `Q17` gagal dengan error `argumen tool tidak valid` (berpotensi menjadi `HTTP 503` ke user) karena model mengirim argumen non-kosong pada tool 0-parameter | Menangkap pelanggaran allowlist/argumen tool di `assistant.go` sebagai penolakan aman dengan penanda `guardrail_blocked: true` (`dapat_ditentukan: false`); uji ulang `Q17` berubah dari FAIL menjadi **PASS** dalam `2.139 ms` |
| 4 | Mengamati perilaku agentic pada `Q16` (*prompt injection*) | Model selalu berhenti di 1 langkah (`cari_dokumen`) | Model menjalankan **3 langkah** (`daftar_sumber` → `lihat_batas_monitoring` → JSON `dapat_ditentukan: false`) | Membuktikan loop `AllowMultiStepToolLoop` aktif dan guardrails prompt+validasi menolak instruksi palsu |

## 3. Keputusan Rancangan Utama
| Keputusan | Alternatif yang Ditolak | Alasan Memilih |
|---|---|---|
| Memisahkan deteksi kamera & tiket tetap **deterministik** (`incidents/`) | Menyerahkan keputusan status kamera/tiket ke agen AI | Penentuan kamera `offline`/pulih 5 menit dan antrean Telegram menuntut kepastian aturan, bebas halusinasi, dan tetap butuh otorisasi Admin manusia (*human-in-the-loop*). |
| Cache vektor dokumen di memori berdasarkan `SHA-256(ID + Text)` | Menghitung ulang embedding seluruh korpus tiap pertanyaan | Menghilangkan pengiriman ulang ~9.885 karakter korpus statis (`8` cache hit per pertanyaan setelah kueri pertama). |
| Batas iterasi agent `MaxSteps = 4` + penangkapan error tool sebagai guardrail | Loop tool tak terbatas atau langsung mengembalikan HTTP 503 | Mencegah pemborosan token akibat loop tak berujung dan menjaga respons aman ke pengguna saat model mengirim argumen tool menyimpang. |

## 4. Transparansi Penggunaan AI & Verifikasi
- Bantuan asisten pemrograman AI digunakan untuk merancang ulang UI chat `/ai-assistant`, menyusun query agregasi 7 hari PostgreSQL, mengimplementasikan loop multi-langkah + caching embedding, serta membantu menyusun draf dokumentasi.
- **Hal yang dikoreksi dari hasil implementasi/asumsi awal:** Asumsi awal bahwa `len(args) != 0` cukup dikembalikan sebagai `error` ternyata memunculkan kegagalan `HTTP 503` pada serangan nyata `Q17`, sehingga diubah menjadi blokir guardrail terstruktur; selain itu asumsi batas RPM Gemini mengharuskan penambahan jeda `7s` pada runner evaluasi.
- **Verifikasi:** Seluruh perubahan diverifikasi melalui unit test Go (`go test ./aiassistant/... ./handlers/... ./incidents/... -count=1`), `tsc --noEmit`, build Next.js produksi, dan eksekusi nyata `cmd/ai-evaluate` terhadap gateway Google AI Studio.

## 5. Daftar Periksa Kesesuaian Akhir (Konversi)
- [x] Persetujuan dosen untuk arsitektur & batas maksimum kelas konversi (`8` dokumen, `3` tool, `17` kasus uji, mode `penuh` + `ringkas`)
- [x] Produk berjalan end-to-end di produksi (`https://nexaspace.site/ai-assistant`) beserta *disclaimer* AI di UI
- [x] Caching embedding + pencatatan pemakaian token `UsageStats` terimplementasi dan teruji
- [x] Evaluasi nyata 17 kasus dengan `gemini-3.1-flash-lite` + perbaikan guardrail `Q17` terdokumentasi di `laporan-evaluasi.md`
- [x] Dokumen `catatan-pemakaian.md`, `guardrails.md`, dan `kajian-risiko.md` disusun berdasarkan hasil uji nyata
