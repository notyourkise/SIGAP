# Issue GitHub hasil audit SIGAP

Seluruh 25 issue dibuat dan dibaca ulang untuk memverifikasi judul, isi, status open dan label. Prioritas: **7 High, 15 Medium, 3 Low**. Basis audit: 31 Oktober 2026, commit `edadd8b1cea2eb316b0f26bcb08c9067eb07be13`.

| Temuan | Prioritas | Issue |
|---|---|---|
| H01 | High | [#167 — Jalur video melewati autentikasi aplikasi](https://github.com/notyourkise/SIGAP/issues/167) |
| H02 | High | [#168 — Isolasi site berlaku pada daftar, tetapi lolos pada akses objek](https://github.com/notyourkise/SIGAP/issues/168) |
| H03 | High | [#169 — Teknisi dapat menggunakan stok pada tugas orang lain](https://github.com/notyourkise/SIGAP/issues/169) |
| H04 | High | [#170 — Mengganti lampiran bisa menghapus file di luar folder unggahan](https://github.com/notyourkise/SIGAP/issues/170) |
| H05 | High | [#171 — Pemakaian stok bersamaan menghasilkan lost update](https://github.com/notyourkise/SIGAP/issues/171) |
| H06 | High | [#172 — Event pemeliharaan disiarkan lintas site dan pemilik](https://github.com/notyourkise/SIGAP/issues/172) |
| H07 | High | [#173 — Tugas dapat dinyatakan selesai memakai checklist karangan](https://github.com/notyourkise/SIGAP/issues/173) |
| M01 | Medium | [#174 — Waktu dari form pemeliharaan bergeser delapan jam](https://github.com/notyourkise/SIGAP/issues/174) |
| M02 | Medium | [#175 — Kebijakan SLA tidak dapat dihubungkan ke jadwal lewat API normal](https://github.com/notyourkise/SIGAP/issues/175) |
| M03 | Medium | [#176 — Pekerjaan yang belum jatuh tempo menurunkan kepatuhan SLA](https://github.com/notyourkise/SIGAP/issues/176) |
| M04 | Medium | [#177 — Zona baru hilang dari daftar site yang sedang dipilih](https://github.com/notyourkise/SIGAP/issues/177) |
| M05 | Medium | [#178 — Kalender mengambil tanggal yang berbeda dari tanggal pengelompokan](https://github.com/notyourkise/SIGAP/issues/178) |
| M06 | Medium | [#179 — Filter tanggal laporan pengecekan lapangan diabaikan](https://github.com/notyourkise/SIGAP/issues/179) |
| M07 | Medium | [#180 — Jadwal bulanan bisa melewati bulan; jadwal berulang juga bergeser](https://github.com/notyourkise/SIGAP/issues/180) |
| M08 | Medium | [#181 — Penyelesaian tugas belum terlindungi dari request bersamaan](https://github.com/notyourkise/SIGAP/issues/181) |
| M09 | Medium | [#182 — Idempotensi generator jadwal belum aman untuk dua worker](https://github.com/notyourkise/SIGAP/issues/182) |
| M10 | Medium | [#183 — Audit log memakai ID objek sebagai ID pengguna pelaku](https://github.com/notyourkise/SIGAP/issues/183) |
| M11 | Medium | [#184 — Penugasan teknisi bisa menciptakan tugas yang tidak dapat diakses](https://github.com/notyourkise/SIGAP/issues/184) |
| M12 | Medium | [#185 — Wajib ganti password ditegakkan oleh UI, belum oleh API](https://github.com/notyourkise/SIGAP/issues/185) |
| M13 | Medium | [#186 — Auto-login/refresh mobile membaca bentuk respons yang salah](https://github.com/notyourkise/SIGAP/issues/186) |
| M14 | Medium | [#187 — Klaim offline-first mobile belum didukung alur penyimpanan/sinkronisasi](https://github.com/notyourkise/SIGAP/issues/187) |
| M15 | Medium | [#188 — Relasi CCTV pada respons pemeliharaan membocorkan field RTSP mentah](https://github.com/notyourkise/SIGAP/issues/188) |
| L01 | Low | [#189 — Tanggal update tidak valid dibalas sukses](https://github.com/notyourkise/SIGAP/issues/189) |
| L02 | Low | [#190 — Koordinat NaN diterima sebagai lokasi valid](https://github.com/notyourkise/SIGAP/issues/190) |
| L03 | Low | [#191 — Label site default tidak selalu mencerminkan scope data](https://github.com/notyourkise/SIGAP/issues/191) |

M15 sebelumnya H08; prioritas sudah diperbarui setelah pemeriksaan database read-only. Tidak ada assignee atau milestone yang ditetapkan. Label yang dipakai berasal dari label existing repo; tidak dibuat label baru. Tidak ada kode aplikasi, database, commit, push atau deployment yang diubah pada proses pembuatan issue.

