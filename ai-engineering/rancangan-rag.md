# Rancangan RAG SIGAP — Minggu 7–9

## Masalah dan batas

Admin dan teknisi membutuhkan cara menemukan informasi dalam dokumentasi SIGAP. Model membantu menjelaskan sumber, sementara status CCTV dan keputusan tiket harus ditentukan oleh sistem operasional dan manusia. Jawaban tidak menggantikan SOP atau menunjukkan kondisi perangkat secara real time.

## Sumber dan izin

`backend/aiassistant/corpus.json` menyimpan potongan awal dari delapan file Markdown yang sudah ada di repositori SIGAP, dengan ID berupa path sumber dan `#inicio`. Periksa daftar dan teks sumber aktual pada file tersebut. Kedelapannya berstatus dokumen proyek; sebagian adalah laporan atau hasil uji historis, bukan SOP terbaru. Jumlah halaman tidak berlaku untuk Markdown. Pemilik proyek tetap perlu memastikan izin pemakaian dan sensitivitas sebelum mengirim potongan kepada gateway eksternal.

Potongan ini adalah snapshot; perubahan file asli tidak otomatis memperbarui corpus. `TestCorpusHasEightDistinctVerifiableSourceSnapshots` memeriksa kesesuaian kutipan dan sumber pada checkout saat tes dijalankan. Delapan file memenuhi angka maksimum pada rumus jumlah minimum menurut K, tetapi kelayakan isi tiap dokumen dan kebutuhan aktual menurut K masih perlu dikonfirmasi.

## Keputusan dan alasan

1. **Pemotongan:** satu potongan awal berukuran sampai 1.250 karakter per sumber. Mudah diaudit untuk prototipe, tetapi belum mencakup keseluruhan dokumen dan dapat memotong konteks.
2. **Embedding:** gateway kompatibel endpoint embeddings, dipilih melalui `AI_EMBEDDING_MODEL`. Kecocokan model dan biaya harus diuji dengan penyedia yang dikonfigurasi.
3. **Retrieval:** cosine similarity, ambil maksimal empat potongan. Implementasi menghitung embedding ulang tiap pertanyaan; ini sederhana tetapi berdampak pada biaya dan latensi.
4. **Konteks:** tool baca-saja mengembalikan ID sumber dan isi potongan, sebagai data yang tidak boleh dianggap instruksi.
5. **Output:** JSON `jawaban`, `dapat_ditentukan`, dan `bukti` berisi ID sumber serta kutipan persis. Validasi memeriksa kutipan muncul dalam teks yang benar-benar diterima model melalui tool. Itu tidak menjamin seluruh makna jawaban setia pada sumber.
6. **Penolakan:** tidak ada korpus atau model menyatakan tidak dapat menentukan menghasilkan jawaban tidak tahu; kutipan fiktif juga menyebabkan penolakan. Model masih dapat memberikan klaim yang salah sambil mencantumkan kutipan nyata, sehingga perlu evaluasi semantik.

Contoh konteks tool: `[{"id":"docs/01-documentation/ACCESS.md#inicio","text":"# Panduan Akses Web SIGAP ..."}]`. Instruksi model aktual ada di `backend/aiassistant/prompts/v2.txt`.

## Evaluasi dan tantangan

Sepuluh pertanyaan ada pada `evaluasi/pertanyaan.json`: enam satu sumber, dua lintas sumber, dua tidak ada dalam sumber. Jalankan `go run ./cmd/ai-evaluate` dari `backend` setelah mengonfigurasi gateway; hasil JSON keluar di stdout dan memerlukan penilaian manusia terhadap ketepatan makna. Perintah memakai model nyata dan dapat menimbulkan biaya. Tidak ada hasil nyata yang diklaim pada dokumen ini.

Pada pertanyaan lintas dokumen, top-k dapat melewatkan satu bagian karena kemiripan vektor atau pemotongan konteks. Dua alternatif: menaikkan top-k (biaya token dan risiko konteks tidak relevan meningkat), atau membagi pertanyaan lalu mengambil sumber untuk setiap subpertanyaan (lebih banyak panggilan dan latensi). Efektivitas keduanya belum diuji pada model nyata.

Tantangan Minggu 9 meminta perbaikan retrieval dan kesetiaan sumber dengan bukti sebelum–sesudah. Belum ada hasil model nyata; jangan mengisi hasil atau metrik secara fiktif. Validasi kutipan dan uji sintetis yang ada adalah bukti kode, bukan bukti kualitas jawaban model.
