# Rancangan sistem SIGAP untuk UTS AI Engineering

Status: bahan rancangan; bukan bukti bahwa presentasi, peer review, atau penilaian UTS sudah dilakukan.

1. **Persoalan:** admin dan teknisi membutuhkan akses cepat ke dokumentasi operasional dan penanganan gangguan CCTV. Pengecekan kondisi sistem dan penyelesaian pekerjaan tetap menjadi tanggung jawab proses SIGAP dan petugas.
2. **Mengapa AI generatif:** menjelaskan kutipan dokumentasi berdasarkan pertanyaan bahasa alami. Penentuan offline, perhitungan waktu, otorisasi, dan tiket memakai aturan serta database karena memerlukan hasil terukur.
3. **Alur:** pengguna login → pertanyaan → embedding gateway → pencarian cosine pada snapshot dokumen → model memilih tool baca-saja → hasil tool → model menyusun JSON → server memvalidasi sumber/kutipan → website menampilkan jawaban.
4. **Kendali output:** schema memiliki `jawaban`, `bukti`, `dapat_ditentukan`. Kutipan harus cocok dengan sumber yang benar-benar dikeluarkan tool. Output tidak valid ditolak; penilaian semantik masih diperlukan.
5. **Tool:** `cari_dokumen`, `lihat_batas_monitoring`, `daftar_sumber`; semuanya baca-saja terhadap korpus statis. Tool tidak menugaskan, menutup tiket, atau mengubah data. Aksi operasional menggunakan izin aplikasi dan verifikasi admin.
6. **Grounding:** delapan sumber dan keputusan retrieval dijelaskan pada `rancangan-rag.md`; sumber dapat kedaluwarsa. Jika bukti tidak dapat diverifikasi, sistem menolak menjawab.
7. **Bukti saat ini:** tes otomatis memakai gateway tiruan, termasuk skenario kutipan valid, kutipan palsu, sumber salah, dan tool yang hanya mengembalikan daftar ID. Keberhasilan atau kegagalan model nyata belum dibuktikan; jangan menyajikan hasil mock sebagai hasil model nyata.
8. **Risiko dan rencana:** dokumen historis, prompt injection dalam sumber, halusinasi meskipun kutipan valid, kebocoran data ke gateway, biaya, dan ketergantungan gateway. Tindak lanjut: validasi sumber, ukur kualitas model nyata, batasi penggunaan data sensitif, lanjut ke materi agent multi-langkah dan evaluasi.

Presentasi UTS menurut modul: maksimal delapan slide, delapan menit presentasi dan tujuh menit tanya jawab. Tanggal pengumpulan dan bukti presentasi mengikuti dosen; belum dicatat selesai.
