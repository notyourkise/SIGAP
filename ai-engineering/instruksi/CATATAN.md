# Catatan versi instruksi

| Versi | Tanggal | Perubahan | Alasan | Efek yang terbukti |
|---|---|---|---|---|
| v1 | 2026-11-02 | Instruksi ringkas; lihat `backend/aiassistant/prompts/v1.txt`. | Baseline rancangan. | Belum ada bukti pengujian model nyata. |
| v2 | 2026-11-02 | Meminta tool, JSON, kutipan persis, dan penolakan tanpa dasar; lihat `backend/aiassistant/prompts/v2.txt`. | Grounding dan validasi terstruktur. | Tes gateway tiruan menguji alur dan validasi; perubahan perilaku model nyata belum dibuktikan. |

Tantangan Minggu 4 (satu kalimat berubah pada sedikitnya tiga dari lima input) belum dapat diklaim selesai: perlu eksperimen v1/v2 dengan model nyata, bukti sebelum-sesudah, dan penjelasan pengguna. Jangan menganggap v1 telah diuji hanya karena file baseline tersedia.
