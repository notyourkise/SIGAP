# Manual Test - UX Filter dan Pencarian Log CCTV

| Item | Detail |
|------|--------|
| **Issue** | #143 (P1 - Perbaikan filter dan pencarian Log CCTV) |
| **Branch** | `feat/cctv-log-ux-filters` |
| **Versi Frontend** | 1.55.37 (setelah fix) |
| **Versi Backend** | 0.64.35 (tidak berubah) |
| **Tanggal** | 2026-10-27 |

---

## 1. Prasyarat

- [ ] Stack berjalan: `db`, `redis`, `api`, `mediamtx`, `nginx`, `frontend`.
- [ ] Tersedia akun **admin** atau teknisi yang memiliki akses ke halaman Log CCTV.
- [ ] Site aktif memiliki data `cctv_stream_logs` (minimal beberapa baris).
- [ ] Untuk uji paginasi, tersedia lebih dari 15 baris log.
- [ ] DevTools browser terbuka pada tab **Network** dengan filter `cctv-logs`.
- [ ] URL halaman `/cctv/logs` terlihat (address bar) untuk memeriksa query string.

Untuk memastikan ada event `unknown`, seed satu baris uji (opsional jika probe
sudah pernah menghasilkan `unknown`):

```sql
INSERT INTO cctv_stream_logs (
    cctv_id, event_type, previous_state, current_state,
    trigger_source, network_type, message
)
SELECT id, 'unknown', '', 'unknown', 'probe', 'gsm', 'Uji manual filter unknown'
FROM cctvs
WHERE deleted_at IS NULL
ORDER BY kode_cctv
LIMIT 1;
```

Setelah pengujian, hapus data uji:

```sql
DELETE FROM cctv_stream_logs WHERE message = 'Uji manual filter unknown';
```

---

## 2. Pengujian Otomatis

Dijalankan dari `frontend/`:

```bash
npm run lint
npx eslint src/app/cctv/logs/page.tsx
npm run build
```

**Kriteria lulus:** `npm run lint` 0 error dan `npm run build` PASS, termasuk
route `/cctv/logs` berhasil diprerender. Warning lama di luar perubahan ini
boleh tetap muncul.

Verifikasi kontrak backend untuk event `unknown` (opsional):

```bash
curl -s \
  -H "Authorization: Bearer <TOKEN>" \
  "http://localhost/api/cctv-logs?event_type=unknown&limit=100"
```

**Kriteria lulus:** respons `200`, seluruh `rows[].event_type` bernilai
`unknown`, dan `summary.total_events` sama dengan jumlah baris yang dikembalikan
(atau total paginasi bila melebihi `limit`).

---

## 3. Skenario Uji

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 1 | Kosongkan pencarian, lalu ketik `THD-12` per karakter dengan jeda ±100 ms. Berhenti mengetik dan tunggu ±1 detik. | Tab Network hanya menampilkan **satu** request `cctv-logs` setelah berhenti mengetik (bukan satu request per karakter). Request memuat `search=THD-12`, dan URL berubah menjadi `?search=THD-12`. | |
| 2 | Dengan lebih dari 15 log, buka halaman 2 melalui paginasi. Lalu ketik kata kunci baru dan tunggu debounce. | Request berikutnya memakai `page=1`, tombol halaman 1 aktif, dan parameter `page` hilang dari URL. | |
| 3 | Klik filter **Tidak Diketahui**. | URL memuat `event_type=unknown`; request `cctv-logs` memuat `event_type=unknown`; seluruh baris menampilkan badge **Tidak Diketahui**; kartu "Total hasil filter" konsisten dengan jumlah data. | |
| 4 | Pada kondisi filter `unknown` aktif, tekan **F5** (refresh). | Filter **Tidak Diketahui** tetap aktif, daftar tetap terfilter, dan URL masih memuat `event_type=unknown`. | |
| 5 | Isi pencarian (mis. `THD-12`), pilih filter **Online**, lalu salin URL dan buka di tab baru. | Tab baru langsung menampilkan filter Online dan pencarian `THD-12` yang sama; nilai input pencarian ikut terisi. | |
| 6 | Ganti beberapa filter/pencarian, lalu tekan tombol **Back** dan **Forward** browser. | Filter, halaman, dan isi input pencarian mengikuti URL pada riwayat tersebut. | |
| 7 | Klik tombol **X** di dalam kolom pencarian. | Input kosong, parameter `search` hilang dari URL, halaman kembali ke 1, dan daftar menampilkan semua event sesuai filter event yang aktif. | |
| 8 | Hentikan container API (`docker compose stop api`), lalu klik **Segarkan**. | Muncul banner "Gagal memperbarui data" dengan tombol **Coba lagi**; daftar lama **tetap terlihat** (tidak diganti skeleton); teks "Terakhir diperbarui" tidak berubah. | |
| 9 | Jalankan kembali API (`docker compose start api`), lalu klik **Coba lagi** pada banner. | Banner hilang tanpa reload halaman, daftar dan ringkasan diperbarui, indikator loading berhenti, dan "Terakhir diperbarui" berubah ke waktu terbaru. | |
| 10 | Muat ulang halaman saat API masih berhenti (uji error awal). | Empty state menampilkan "Gagal memuat log status" beserta pesan error dan tombol **Coba lagi**; tidak menampilkan "Belum ada catatan log status". Setelah API hidup, **Coba lagi** memuat data tanpa reload penuh. | |
| 11 | Cari teks yang tidak ada (mis. `ZZZ-TIDAK-ADA`). | Empty state menampilkan **Tidak ada hasil filter**, menyebut kata kunci, dan menyediakan tombol **Hapus filter**. Klik tombol tersebut mengosongkan filter/pencarian dan memuat daftar normal. | |
| 12 | Pilih site yang belum memiliki log sama sekali (tanpa filter/pencarian). | Empty state menampilkan **Belum ada catatan log status** dengan penjelasan bahwa riwayat akan tersimpan setelah probe/verifier berjalan. | |
| 13 | Klik **Segarkan** saat daftar sudah berisi data. | Daftar lama tetap tampil; header menampilkan "Memperbarui…" dengan spinner; setelah selesai berubah menjadi "Terakhir diperbarui <tanggal/jam>". Skeleton hanya muncul pada pemuatan awal. | |
| 14 | Ubah halaman ke 2, lalu refresh halaman. | URL memuat `page=2`, paginasi tetap di halaman 2, dan daftar menampilkan baris ke-16 dan seterusnya. | |
| 15 | Periksa aksesibilitas keyboard: Tab ke filter dan tombol hapus pencarian, lalu aktifkan dengan Enter/Space. | Fokus terlihat, filter berubah, dan `aria-pressed` menandai filter aktif. | |

---

## 4. Kriteria Lulus

- [ ] Mengetik pencarian tidak membanjiri API: hanya satu request setelah jeda 300 ms.
- [ ] Filter `unknown` / "Tidak Diketahui" berfungsi dan konsisten dengan ringkasan.
- [ ] Refresh mempertahankan filter, pencarian, dan halaman melalui URL.
- [ ] Error dapat dipulihkan dengan tombol **Coba lagi** tanpa reload penuh.
- [ ] Data lama tidak tertutup skeleton saat refresh; "Terakhir diperbarui" tampil.
- [ ] Empty state "belum ada log" dan "tidak ada hasil filter" dibedakan.
- [ ] `npm run lint` 0 error dan `npm run build` PASS.

---

## Catatan

- Tidak ada perubahan backend pada issue ini. Endpoint `/cctv-logs` sudah
  melakukan exact match untuk `event_type`, termasuk nilai `unknown`.
- Jika data `unknown` belum ada, gunakan SQL seed pada bagian prasyarat dan
  jangan lupa menghapusnya setelah pengujian.
- Untuk menguji error awal (skenario 10), lakukan pada tab baru agar data lama
  di tab utama tidak mengaburkan hasil.
