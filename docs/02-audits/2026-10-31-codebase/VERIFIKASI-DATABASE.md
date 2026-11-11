# Verifikasi lanjutan database SIGAP

Tanggal: **31 Oktober 2026, 23.00 WITA**. Melanjutkan audit setelah pengguna memberi tahu Docker sudah dijalankan.

## Hasil utama

Database nyata berhasil diperiksa tanpa menulis data operasional. Daftar tetap **25 temuan**, tetapi penilaian RTSP lama **H08 diturunkan menjadi M15 (Medium)** setelah diketahui 35 URL CCTV saat ini semuanya memakai penanda format terenkripsi `enc:v1:` dan tidak ada URL legacy plaintext. Ringkasan terbaru: **7 High, 15 Medium, 3 Low**.

Bug penyaringan respons RTSP masih terbukti oleh test kode sebelumnya. Yang dikoreksi adalah tingkat dampak pada data saat ini: audit tidak menemukan password plaintext pada kolom URL tersebut. Pemeriksaan format tidak mendekripsi data atau membuktikan validitas kriptografis setiap ciphertext.

## Batas tindakan dan lingkungan

- Docker Desktop dapat diakses, tetapi seluruh container SIGAP masih Exited ketika diperiksa. Anto hanya menjalankan `docker start sigap_postgres`, memakai container dan volume existing `sigap_postgres_data`.
- PostgreSQL **15.18** berhasil menerima koneksi dan berstatus healthy.
- API, frontend, Nginx, Redis, backup, monitoring dan MediaMTX tidak dijalankan. Tidak ada startup aplikasi, migrasi, seed, job otomatis, pengiriman notifikasi, koneksi ke kamera, atau pengujian mutasi API.
- Seluruh sesi SQL menggunakan `PGOPTIONS` dengan `default_transaction_read_only=on` dan `statement_timeout=10000`. Pemeriksaan utama juga memakai `BEGIN ... READ ONLY` lalu `ROLLBACK`.
- Nilai password, token, RTSP URL, hash password, nama pengguna, dan email tidak ditampilkan/disalin ke laporan. Hanya agregat, definisi skema, dan fingerprint tabel yang dicatat.
- Pada akhir pemeriksaan hanya container PostgreSQL yang berjalan; Anto membiarkannya berjalan agar database tetap tersedia.

## Data yang ditemukan

| Data | Hasil |
|---|---:|
| Akun aktif admin | 2 |
| Akun aktif teknisi | 3 |
| Teknisi dengan membership site aktif | 3 dari 3 |
| Akun aktif yang masih wajib ganti password | 4: 1 admin dan 3 teknisi |
| Site aktif | 1 |
| CCTV aktif | 35 |
| CCTV tanpa site | 0 |
| CCTV dengan stream_enabled | 35 |
| URL RTSP dengan prefix enkripsi `enc:v1:` | 35 |
| URL RTSP nonkosong tanpa prefix enkripsi | 0 |
| Zona aktif | 6 |
| Zona tanpa site | 0 |
| Template aktif | 9 |
| Template aktif berstatus wajib | 9 |
| Log pemeliharaan, termasuk soft-deleted | 0 |
| Jadwal pemeliharaan, termasuk soft-deleted | 0 |
| Kebijakan SLA, termasuk soft-deleted | 0 |
| Pengecekan lapangan, termasuk soft-deleted | 0 |
| Master suku cadang, termasuk soft-deleted | 0 |
| Riwayat penggunaan suku cadang, termasuk soft-deleted | 0 |

Angka nol membatasi reproduksi dengan data nyata. Audit tidak membuat fixture atau menulis tugas/stok ke database saat ini untuk memaksakan skenario. Hasil test simulasi sebelumnya tetap berlaku sebagai bukti perilaku kode, bukan bukti transaksi bermasalah pernah terjadi pada database ini.

## Dampak pada temuan sebelumnya

| Temuan | Verifikasi lanjutan | Kesimpulan |
|---|---|---|
| H08 -> **M15**: field RTSP mentah pada nested response | Semua 35 URL memakai format enkripsi; URL plaintext 0 | Prioritas sekarang Medium. Risiko password plaintext bersifat kondisional terhadap data legacy, tidak ditemukan pada snapshot ini |
| H02: akses lintas site | Saat ini hanya ada satu site aktif dan seluruh CCTV mempunyai site | Bug kode tetap ada; tidak direproduksi lintas dua site nyata karena site kedua tidak dibuat |
| H03/H05: akses/pengurangan stok | Master stok dan usage kosong | Belum ada bukti selisih stok aktual; tidak ada stok yang dikurangi audit |
| H07: checklist tidak sesuai template | Terdapat 9 template aktif dan seluruhnya wajib | Memperkuat konteks bahwa payload satu item bebas tidak mewakili checklist wajib existing; completion nyata tidak dicoba karena belum ada tugas |
| M04: zona baru kehilangan site | Keenam zona existing memiliki site | Tidak ada zona orphan yang ditemukan; bug terjadi pada jalur pembuatan baru, tidak diuji dengan insert |
| M08: completion bersamaan | Index unik field_maintenances hanya primary key ID; tidak ada unique/exclusion constraint pada maintenance_log_id dan tidak ada trigger aplikasi terdaftar | Skema nyata tidak memberi pengaman duplikasi hasil per tugas. Race API/PostgreSQL belum dieksekusi |
| M09: generator bersamaan | Index unik maintenance_logs hanya primary key ID; tidak ada unique/exclusion constraint kejadian jadwal dan tidak ada trigger aplikasi terdaftar | Skema nyata tidak menutup celah idempotensi kode. Dua worker tidak dijalankan |
| M10: ID objek dikirim sebagai actor audit | FK `fk_audit_logs_user` benar-benar ada, menunjuk users(id) | ID log/site/zona yang bukan ID user akan ditolak FK ketika dipakai sebagai user_id; jalur audit salah diperkuat skema aktual |
| M11: penugasan lintas membership | Ketiga teknisi memiliki membership site aktif; tidak ada tugas/jadwal | Belum ada tugas macet yang ditemukan pada snapshot ini |
| M12: wajib ganti password hanya di UI | Empat akun aktif mempunyai flag true | Prasyarat tersebut ada dalam data nyata; pemanggilan API menggunakan akun nyata tidak dilakukan |

## Kondisi migrasi dan artefak deployment

`schema_migrations` melaporkan **version 28, dirty=false**. Checkout memiliki file sampai `000029_cctv_stream_logs_indexes.up.sql`; ada satu migrasi index yang belum diterapkan pada database ini. Anto tidak menjalankannya. Ini dicatat sebagai perbedaan kesiapan lingkungan, bukan tambahan bug produk ke-26.

Container API yang ditemukan dibuat 23 Oktober 2026, memakai image ID `sha256:3975189059b418c8efe028e42f1e7d90a2b13f3b39318bb05ea5e75ee85c19ce`. Tidak ada verifikasi bahwa binary di image tersebut identik dengan checkout audit. Karena itu container lama tidak dipakai sebagai pembuktian runtime kode terbaru.

Konfigurasi Nginx dan MediaMTX pada container existing di-bind-mount read-only ke file repo yang ditinjau. Ini memperjelas asal konfigurasi H01, tetapi belum membuktikan video benar-benar bisa ditonton tanpa login karena kedua layanan masih berhenti.

## Integritas dan bukti

Fingerprint delapan tabel operasional diperiksa sebelum dan setelah rangkaian pemeriksaan: **semuanya sama**. Tabelnya users, cctvs, zones, maintenance_logs, maintenance_schedules, field_maintenances, spare_parts, dan spare_part_usages. Fingerprint dihitung dari keseluruhan baris di server; data mentah tidak dikeluarkan. Jaminan batas tulis utama berasal dari transaksi read-only; fingerprint adalah bukti tambahan untuk delapan tabel tersebut.

- `database-readonly.sql`: query agregat dan fingerprint, tanpa kredensial.
- `database-readonly.log`: hasil snapshot awal.
- `database-readonly-after.log`: hasil snapshot akhir yang cocok.
- `database-schema-verification.log`: format penyimpanan RTSP, index unik, FK audit, dan pengecekan trigger.

**Rekomendasi:** prioritaskan penutupan akses video/file serta isolasi data terlebih dahulu. Pengujian transaksi nyata untuk stok dan completion perlu memakai fixture terkontrol serta rencana rollback pada pekerjaan berikutnya; jangan menganggap tabel kosong sebagai bukti jalur tersebut bebas bug.
