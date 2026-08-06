# Panduan Uji Manual - Dashboard Kinerja Teknisi (Task 6.2)

## Prasyarat

- Backend dan frontend berjalan pada environment yang sama.
- Tersedia satu akun Admin dan satu akun Teknisi.
- Tersedia log pemeliharaan dengan status `scheduled`, `ongoing`, `completed`, dan/atau `failed`.
- Tersedia catatan pengecekan lapangan dengan hasil `ok` dan `attention`.

## Skenario

| ID | Skenario | Langkah | Hasil yang diharapkan |
|----|----------|---------|----------------------|
| S1 | Akses admin | Login sebagai Admin, buka `/kinerja-teknisi`. | Halaman dapat dibuka dan menu "Kinerja Teknisi" terlihat di bawah Dashboard. |
| S2 | Blokir teknisi | Login sebagai Teknisi, akses `/kinerja-teknisi` secara langsung. | Pengguna diarahkan kembali ke `/dashboard`; endpoint mengembalikan 403 bila dipanggil langsung. |
| S3 | Rentang 7 hari | Klik tombol "7 Hari". | Request mengirim `start_date` dan `end_date`; data tabel dan chart memuat ulang untuk 7 hari terakhir. |
| S4 | Rentang 30 hari | Klik tombol "30 Hari". | Data menampilkan aktivitas 30 hari terakhir dan label rentang berubah. |
| S5 | Rentang kustom | Klik "Kustom", isi tanggal mulai dan tanggal akhir yang valid. | Data memuat ulang sesuai tanggal; tanggal akhir termasuk seluruh hari tersebut. |
| S6 | Ranking dan sorting | Klik judul setiap kolom tabel, termasuk nama, total tugas, rate, overdue, dan kualitas. | Arah sorting berubah; indikator panah menunjukkan kolom dan arah aktif. |
| S7 | Chart dan badge | Bandingkan teknisi dengan completion rate minimal 80%, 50-79%, dan di bawah 50%. | Chart horizontal menampilkan semua teknisi; badge rate berwarna hijau, kuning, dan merah sesuai ambang serta tetap memiliki angka. |
| S8 | Empty/error state | Pilih rentang tanpa aktivitas atau hentikan backend lalu muat ulang. | Tampil empty state yang informatif atau pesan error dengan tombol "Coba lagi"; layout tidak rusak di mobile. |

## Verifikasi API

```text
GET /api/dashboard/technician-performance
GET /api/dashboard/technician-performance?start_date=2026-07-01&end_date=2026-07-31
```

Respons sukses memiliki `status: "success"` dan `data` berupa array. Setiap item memiliki `user_id`, `full_name`, `username`, `total_tasks`, `completed`, `failed`, `ongoing`, `completion_rate`, `avg_completion_hours`, `overdue_count`, `field_check_count`, dan `quality_score`.

## Kriteria Perhitungan

- `completion_rate` = `completed / total_tasks * 100`.
- `avg_completion_hours` dihitung dari `assigned_at` ke `completed_at`; log lama yang `assigned_at`-nya kosong memakai `created_at`.
- `overdue_count` mencakup tugas aktif yang melewati `scheduled_date` dan tugas completed yang selesai setelah `scheduled_date`.
- `quality_score` = pengecekan dengan `overall_status = ok` dibagi total pengecekan lapangan.
