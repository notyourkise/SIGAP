# Catatan proses dan batas bukti

## 2 November 2026

- Menggunakan bantuan AI untuk memetakan kebutuhan SIGAP, merancang monitoring dan membangun prototipe asisten berbasis dokumen. Pengguna bertanggung jawab meninjau serta menjelaskan hasilnya.
- Mengintegrasikan gateway kompatibel embeddings/chat completions, retrieval cosine, tool baca-saja, format JSON dan validasi kutipan pada `backend/aiassistant`.
- Memperluas korpus menjadi delapan snapshot dari file Markdown proyek dengan sumber yang dapat dilacak.
- Menyusun 10 pertanyaan evaluasi grounding pada `evaluasi/pertanyaan.json`, termasuk pertanyaan lintas sumber dan pertanyaan tanpa jawaban.
- Pengujian otomatis mencakup skenario validasi kutipan, korpus dan gateway tiruan. Data model tiruan bukan output model nyata. Lihat hasil tes yang dijalankan pada lingkungan kerja, bukan skor yang direkayasa.
- Gateway/model nyata belum dikonfigurasi. Perbandingan prompt v1/v2, evaluasi output aktual, biaya/token, kegagalan retrieval nyata dan before/after Minggu 4/6/9 belum dibuktikan.
- Proyek kuliah menuntut catatan pribadi, pemahaman, lembar tema, tugas tantangan, peer review dan UTS sesuai modul. File ini adalah awal catatan, bukan klaim pekerjaan pribadi tersebut telah selesai.

## Batas keselamatan dan mutu

Jangan memasukkan kunci, data pribadi, detail jaringan sensitif, atau status perangkat rahasia ke pertanyaan gateway eksternal. Potongan sumber bersifat historis dan harus diperiksa izin serta relevansinya. Memeriksa bahwa kutipan benar-benar ada tidak membuktikan isi kesimpulan model benar.
