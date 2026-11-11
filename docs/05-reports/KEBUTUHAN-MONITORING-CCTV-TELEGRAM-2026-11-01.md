# Kebutuhan Monitoring CCTV, Tiket Otomatis, dan Telegram SIGAP

Tanggal: 1 November 2026 (WITA)

Status: disetujui pengguna sebagai acuan kebutuhan. Bukan desain teknis final, bukti implementasi, atau klaim target sudah tercapai. Keputusan yang masih terbuka dan usulan yang ditandai tetap perlu diselesaikan sebelum implementasi.

Anto menangani audit, diskusi kebutuhan, dan verifikasi. Implementasi serta perbaikan dikerjakan Antigravity. Pembuatan dokumen ini tidak mengubah kode, konfigurasi, database, atau perangkat.

## 1. Tujuan

Teknisi mengetahui CCTV yang offline melalui grup Telegram operasional tanpa terus-menerus memeriksa NVR. SIGAP membuat tiket otomatis berdasarkan kegagalan yang terkonfirmasi. Admin mengatur penugasan dan memverifikasi penutupan pekerjaan melalui website.

Monitoring ini menjadi bagian dari rencana fitur agent di SIGAP. Website tetap menjadi tempat pengelolaan pekerjaan. Cakupan dokumen ini hanya monitoring, tiket, dan pemberitahuan Telegram; kebutuhan agent lainnya belum dirinci di sini.

## 2. Keputusan yang sudah disepakati

| Bagian | Kesepakatan |
|---|---|
| Cakupan kamera | Seluruh CCTV, termasuk Metro-E dan GSM. |
| Target offline | Maksimal 5 menit sejak CCTV benar-benar offline sampai tiket dibuat dan pemberitahuan masuk ke grup Telegram operasional teknisi. |
| Akses teknisi | Teknisi cukup menerima informasi di grup, tanpa menautkan akun Telegram ke SIGAP. |
| Penugasan | Admin melakukan penugasan melalui website SIGAP. |
| Aksi dari Telegram | Mengambil tiket atau mengubah pekerjaan langsung dari grup ditunda. |
| Pemulihan | CCTV harus stabil online selama 5 menit berturut-turut sebelum pemulihan dikonfirmasi. Jika terputus lagi, hitungan dimulai ulang. |
| Penutupan | Koneksi pulih tidak otomatis menutup tiket. Admin memverifikasi dan menutupnya. |
| Batas verifikasi admin | Khusus tiket gangguan otomatis CCTV. Maintenance rutin tetap menggunakan alur yang sekarang; tidak ikut diwajibkan menunggu verifikasi admin. |
| Gangguan berulang | Jika tiket gangguan yang sama masih terbuka, perbarui tiket tersebut, bukan membuat tiket baru. |
| Gangguan bersama | Satu tiket boleh mencakup beberapa CCTV jika ada bukti gangguan pada NVR atau jaringan yang sama. Semua kamera terdampak tetap dicatat. |
| Penyebab belum terbukti | Tiket tetap per kamera; pemberitahuan beberapa kamera dapat digabung. Jangan menyimpulkan NVR rusak tanpa bukti. |
| Pemulihan sebagian | Tiket bersama tetap terbuka. Telegram memperbarui daftar kamera yang masih offline. |
| Penutupan tiket bersama | Seluruh kamera terdampak harus pulih stabil selama 5 menit dan diverifikasi admin. |
| Pemantau gagal | Status kamera menjadi “belum dapat dipastikan”; beri tahu admin. Jangan otomatis membuat banyak tiket kamera offline. |

## 3. Alur operasional

### 3.1 Kamera offline

1. Sistem memperoleh bukti kondisi kamera/kanal yang masih berlaku.
2. Sistem mengonfirmasi kegagalan dan membedakannya dari kegagalan pemantau. Cara konfirmasi teknis masih perlu ditentukan.
3. Sistem mencari tiket terbuka untuk gangguan yang sama.
4. Jika sudah ada, perbarui tiket. Jika belum ada, buat tiket otomatis. Gunakan tiket bersama hanya jika bukti penyebab bersama memadai.
5. Kirim pemberitahuan ke grup operasional. Penggabungan pesan tidak boleh membuat target 5 menit terlewati.
6. Admin menugaskan teknisi melalui website. Grup mendapat pembaruan penanggung jawab.

Jika penyebab belum jelas, gunakan keterangan faktual seperti “CCTV tidak dapat diakses”, tanpa menyatakan perangkat rusak.

### 3.2 Kamera kembali online

1. Saat ada bukti kamera kembali online, mulai pengamatan stabil selama 5 menit berturut-turut.
2. Jika kamera terputus lagi, mulai ulang hitungan. Tiket yang masih terbuka tetap digunakan.
3. Setelah kestabilan terkonfirmasi, kirim kabar pemulihan ke grup dan beri keterangan “koneksi pulih, menunggu verifikasi” pada tiket.
4. Admin memverifikasi kondisi dan menutup tiket jika gangguan selesai.
5. Untuk tiket bersama, catat pemulihan per kamera. Pemulihan sebagian tidak memenuhi syarat penutupan.

Keterangan di atas adalah makna operasional yang disepakati, bukan penetapan nama status atau perubahan skema database.

### 3.3 Pemantau kehilangan akses

1. Tandai kondisi sebagai “belum dapat dipastikan” ketika bukti tidak cukup.
2. Beri tahu admin bahwa pemeriksaan terganggu.
3. Jangan menyimpulkan semua kamera mati hanya karena pemantau tidak menjangkau sumber.
4. Jika pemantau gagal ketika tiket sudah terbuka, pertahankan tiket dan jangan menandainya pulih tanpa bukti baru.

Aturan pada poin 4 adalah rekomendasi Anto untuk menjaga konsistensi, masih perlu konfirmasi. Kanal pemberitahuan khusus admin belum ditentukan.

## 4. Isi pemberitahuan

Kebutuhan informasi yang telah dibahas:

- Kamera atau daftar kamera terdampak dan lokasinya.
- Waktu terdeteksi serta nomor tiket dan tautan ke website SIGAP.
- Penanggung jawab setelah admin melakukan penugasan.
- Kabar pemulihan dan kamera yang masih offline jika pemulihan baru sebagian.

Rekomendasi Anto, belum menjadi keputusan final: sertakan waktu pemeriksaan terakhir dan penjelasan singkat sumber bukti agar teknisi memahami batas informasi. Pesan tidak memuat kata sandi, token, atau URL stream berkredensial. Tautan tiket tetap mengikuti autentikasi dan hak akses SIGAP; keberadaan di grup bukan pemberian akses website.

## 5. Kebijakan monitoring GSM

Mengikuti [kebijakan SIGAP bagian 11.2](../01-documentation/AGENTS.md):

- Larangan mutlak ICMP dan pemeriksaan aktif GSM sudah dicabut atas keputusan pengguna.
- ICMP, pemeriksaan protokol, dan verifikasi stream singkat diperbolehkan secara terbatas.
- Durasi, timeout, percobaan ulang, serta jumlah pemeriksaan bersamaan harus dibatasi. Sesi monitoring harus benar-benar ditutup.
- Background stream terus-menerus tanpa batas tidak menjadi metode monitoring default.
- Penggunaan data aktual harus diukur; jangan mengklaim konsumsi nol atau angka tertentu tanpa pengukuran.
- Paket/kuota GSM dan model NVR belum diketahui. Hal ini tidak membatalkan izin monitoring aktif, tetapi membatasi pilihan integrasi yang dapat dibuktikan saat ini.
- Ping NVR yang berhasil tidak membuktikan seluruh kanal kameranya sehat. Stream on-demand yang sedang tidak dipakai juga tidak membuktikan kamera offline.

## 6. Pilihan pendekatan dan statusnya

| Pendekatan yang dibahas | Konsekuensi | Keputusan |
|---|---|---|
| Telegram untuk pemberitahuan; penugasan di website | Teknisi tidak perlu menautkan akun. Identitas penugasan dikelola di SIGAP. | Dipilih untuk tahap awal. |
| Teknisi mengambil tiket dari grup | Perlu mengenali identitas Telegram dan memeriksa haknya; bisa melalui penautan atau pemetaan oleh admin. | Ditunda karena menambah pengaturan awal. |
| Tutup otomatis setelah koneksi pulih | Tidak memerlukan verifikasi admin, tetapi dapat menutup gangguan yang belum benar-benar selesai. | Tidak dipilih. |
| Konfirmasi pemulihan lalu verifikasi admin | Tiket tetap terbuka sampai admin memastikan penyelesaian. | Dipilih. |

Keputusan apakah memakai layanan NVR, pemeriksaan langsung, MediaMTX, atau gabungan beberapa sumber belum ditetapkan. Dokumen ini tidak memilih pustaka, penyedia AI, atau arsitektur worker.

## 7. Batas peran AI — rekomendasi Anto

Bagian ini usulan, belum keputusan rinci pengguna:

- Monitoring dan aturan teruji menentukan status, waktu, penggabungan gangguan, serta pembuatan tiket.
- AI dapat membantu merangkum bukti, menyusun uraian tiket, dan menyarankan langkah pemeriksaan berdasarkan SOP.
- Dugaan AI tidak menjadi bukti tunggal bahwa kamera offline atau beberapa kamera memiliki penyebab yang sama.
- Pemanggilan AI jangan menjadi ketergantungan wajib yang menghalangi tiket dan notifikasi memenuhi target 5 menit.

## 8. Keputusan teknis yang masih terbuka

Antigravity perlu menyelesaikan desain untuk bagian berikut, dan mengembalikan pertanyaan yang mengubah proses bisnis kepada pengguna sebelum implementasi:

1. Sumber bukti per kamera, model/protokol NVR yang tersedia, serta keterjangkauan jaringan dari lokasi pemantau.
2. Kriteria kegagalan terkonfirmasi, interval pemeriksaan, batas waktu, percobaan ulang, kapasitas pemeriksa, dan pembagian anggaran waktu total maksimal 5 menit.
3. Definisi “gangguan yang sama”, bukti untuk tiket bersama, dan penanganan jika penyebab bersama baru ditemukan setelah tiket per kamera dibuat. Jangan otomatis menggabungkan atau menutup tiket lama tanpa aturan yang disepakati.
4. Definisi stabil selama 5 menit berdasarkan pemeriksaan berkala: jarak antar-bukti, kegagalan pemeriksaan, dan celah pengamatan. Lima menit berlalu tanpa bukti baru tidak cukup membuktikan kestabilan.
5. Penyimpanan riwayat kejadian dan pencegahan tiket ganda ketika pemeriksaan berjalan bersamaan, layanan dimulai ulang, atau pengiriman dicoba lagi.
6. Grup tujuan, pengelolaan bot dan rahasianya, hak bot, kanal khusus admin, serta mekanisme percobaan ulang dan eskalasi saat Telegram gagal.
7. Jeda penggabungan pesan, pembaruan pesan, dan pengingat untuk gangguan yang belum ditangani. Belum ada angka atau aturan pengingat yang disepakati.
8. Penyesuaian model tiket yang ada untuk beberapa CCTV, status operasional, audit tindakan admin, dan isolasi akses antar-site. Kompatibilitas kode belum diperiksa dalam dokumen ini.
9. Batas konsumsi data GSM dan kapasitas monitoring setelah pengukuran nyata. Angka batas belum ditetapkan.
10. Gangguan baru setelah tiket lama ditutup, kamera yang sengaja dinonaktifkan, dan periode maintenance terjadwal. Pengecualian dari target 5 menit belum disepakati.

## 9. Usulan skenario penerimaan untuk implementasi

Skenario di bawah adalah kriteria verifikasi yang diusulkan Anto berdasarkan kebutuhan, bukan hasil tes. Pengujian gangguan dilakukan pada lingkungan uji yang diizinkan, tanpa mematikan perangkat operasional atau merusak database yang ada.

| Skenario | Hasil yang perlu dibuktikan |
|---|---|
| Satu kamera benar-benar offline | Tiket dan pesan grup tersedia dalam maksimal 5 menit sejak waktu gangguan yang diketahui pada pengujian. Uji Metro-E dan GSM. |
| Kamera terakhir diperiksa tepat sebelum gangguan | Kasus waktu deteksi terburuk tetap memenuhi target; pengukuran tidak dimulai hanya setelah sistem menyadari kegagalan. |
| Ping NVR berhasil, satu kanal gagal | Kanal bermasalah tidak dianggap sehat hanya karena NVR dapat dijangkau. |
| Tidak ada penonton stream on-demand | Kamera tidak otomatis dinyatakan offline hanya karena stream idle. |
| Pemeriksaan serentak atau layanan dimulai ulang | Gangguan sama yang masih terbuka tidak menghasilkan tiket ganda. |
| Pengiriman Telegram gagal | Tiket tidak digandakan; kegagalan dan percobaan ulang tercatat. Jika melewati 5 menit, laporkan target gagal, bukan dianggap lulus. |
| Kamera online kurang dari 5 menit lalu offline lagi | Tidak diumumkan pulih terkonfirmasi; hitungan dimulai ulang dan tiket terbuka yang sama diperbarui. |
| Kamera stabil online 5 menit berturut-turut | Pesan pemulihan dikirim, tiket menunggu verifikasi, dan tidak otomatis ditutup. |
| Gangguan bersama terbukti | Satu tiket mencatat seluruh kamera terdampak; pesan diringkas tanpa melampaui target awal. |
| Banyak kamera gagal, penyebab bersama belum terbukti | Tiket per kamera tetap tersedia dan pesan dapat digabung; tidak ada kesimpulan NVR rusak tanpa bukti. |
| Tiket bersama baru pulih sebagian | Tiket tetap terbuka, kamera yang masih offline tetap tercatat, dan grup mendapat pembaruan. |
| Semua kamera pada tiket bersama pulih | Masing-masing memenuhi kestabilan 5 menit, lalu admin memverifikasi sebelum penutupan. |
| Pemantau kehilangan akses atau bukti kedaluwarsa | Status belum dapat dipastikan, admin diberi tahu, dan tidak terjadi pembuatan massal tiket kamera offline yang keliru. |
| Teknisi belum menautkan akun Telegram | Tetap dapat melihat pemberitahuan grup; admin dapat menugaskan lewat website. |
| Pengukuran GSM | Ada catatan durasi sesi, penutupan sumber, dan konsumsi data aktual beserta batas pengukurannya. |

Rekomendasi bukti uji: waktu gangguan, waktu pengamatan, hasil dan sumber bukti, waktu tiket dibuat, upaya pengiriman, serta waktu pesan muncul di grup. Respons sukses API Telegram perlu dibedakan dari bukti pesan tampil di grup dan dari bukti teknisi membacanya. Target yang disepakati adalah pesan masuk ke grup, bukan seluruh anggota telah membacanya.

## 10. Tindak lanjut

1. Rangkuman sudah disetujui pengguna sebagai acuan. Usulan yang ditandai dan keputusan terbuka yang memengaruhi operasional masih perlu diselesaikan.
2. Antigravity memeriksa implementasi SIGAP saat ini, menyusun desain teknis dan rencana pengujian, lalu mengklarifikasi keputusan bisnis yang masih terbuka.
3. Implementasi baru dikerjakan sesuai otorisasi pengguna kepada Antigravity. Dokumen ini tidak memberi izin perubahan database operasional atau deployment.
4. Anto melakukan audit/verifikasi berdasarkan bukti implementasi dan pengujian yang tersedia; klaim target 5 menit menunggu hasil uji.
