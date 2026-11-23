# Catatan Versi Instruksi (System Prompt)

| Versi | Tanggal | Perubahan | Alasan | Efek yang Terbukti |
|---|---|---|---|---|
| v1 | 2026-11-02 | Instruksi ringkas bebas tanpa kontrak tool maupun JSON (`backend/aiassistant/prompts/v1.txt`). | Baseline awal rancangan. | Tidak menjamin format JSON ataupun kutipan sumber persis. |
| v2 | 2026-11-02 | Mewajibkan pemanggilan `cari_dokumen`, melarang bukti dari `daftar_sumber`, menetapkan tepat 3 key JSON wajib, dan mewajibkan penolakan bila sumber tidak cukup (`backend/aiassistant/prompts/v2.txt`). | Menegakkan structured output, grounding, dan validasi kutipan verbatim di `Validate()`. | Pada evaluasi awal 10 kasus dan evaluasi bersih 17 kasus dengan `gemini-3.1-flash-lite`, seluruh kasus valid yang terjawab gateway mengembalikan format JSON valid dan lolos `Validate()`. |
| v2 + Mode Ringkas | 2026-11-23 | Menambahkan instruksi dinamis saat `AnswerOptions.Mode == "ringkas"` agar `jawaban` dibatasi maksimal 2–3 kalimat padat atau 3 poin singkat sambil mempertahankan `bukti` dan `dapat_ditentukan`. | Memenuhi syarat fleksibilitas output ringkas & penuh tanpa mengganggu baseline mode penuh. | Pada perbandingan 15 kasus lolos yang sama, total token output turun dari `3.363` (`penuh`) menjadi `3.031` (`ringkas`); pada kueri multi-sumber `Q08`, token output turun dari `617` menjadi `340`. |
