# Audit kode SIGAP — 31 Oktober 2026 (WITA)

Audit oleh Anto atas permintaan pengguna. Hasil: **25 temuan — 7 High, 15 Medium, 3 Low**. Sebanyak **17 temuan direproduksi secara terkontrol**; **8 temuan berbasis penelusuran kode/konfigurasi**. Tidak ada perbaikan kode aplikasi, migrasi, seed, perubahan database operasional, commit, push, atau deployment.

## Cakupan dan tingkat bukti

- Checkout: `D:\laragon\www\SIGAP`, branch `main`, HEAD `edadd8b1cea2eb316b0f26bcb08c9067eb07be13`.
- Role aplikasi: **admin** dan **teknisi**, berdasarkan `docs/01-documentation/PRODUCT.md` dan `AGENTS.md` di direktori dokumentasi. Akses anonim dan API key diperiksa sebagai jalur autentikasi tambahan, bukan role pengguna baru.
- Acuan: dokumentasi repo, registrasi route, middleware, model/migrasi, pemanggil di web/mobile, serta pengujian kode. Tidak tersedia SOP bisnis independen; penilaian proses bisnis memakai perilaku yang dijanjikan dokumentasi dan konsistensi antarfitur.
- Modul yang ditinjau: autentikasi/profil/pengguna, CCTV/QR/impor, pemeliharaan/checklist, jadwal/kalender, stok, zona/site, SLA, dashboard/kinerja/digest, laporan, notifikasi/email/push/WebSocket, API key/webhook, streaming/snapshot, dan integrasi mobile. Kedalaman reproduksi berbeda per modul; tabel di bawah menjelaskannya.
- Tool knowledge graph tidak tersedia pada sesi ini. Penelusuran dilakukan melalui sumber kode langsung; tidak ada klaim bahwa graph atau `check_index_coverage` berhasil digunakan.
- Docker gagal diakses: pipe `dockerDesktopLinuxEngine` tidak tersedia. Karena itu database saat ini, konfigurasi container yang benar-benar aktif, streaming kamera, SMTP, push nyata, dan mobile di perangkat **tidak diverifikasi langsung**. Anto tidak menyalakan aplikasi karena startup menjalankan migrasi/job otomatis.
- Pengujian handler menggunakan `sqlmock` dan identitas role/site yang diinjeksi, kecuali test middleware yang menggunakan JWT uji. Hasil ini membuktikan perilaku kode pada fixture, bukan keberadaan data/kasus tersebut di database saat ini.
- Test WebSocket memakai koneksi nyata hanya pada `127.0.0.1` dengan port sementara. Test penghapusan file memakai satu file penanda baru milik audit, bukan file pengguna.
- Perubahan pengguna yang sudah ada sejak awal: `frontend/src/components/CCTVMap.tsx` dan dokumen arsip CCTV. Keduanya tidak diedit oleh audit.

**Klasifikasi:** High = kebocoran akses/data, penghapusan file, integritas stok, atau bypass hasil pemeriksaan; Medium = alur operasional/perhitungan/penjadwalan tidak konsisten atau rapuh; Low = validasi atau penyajian konteks yang dampaknya lebih terbatas. Prioritas mempertimbangkan potensi dampak, bukan klaim insiden yang sudah terjadi.

**Label bukti:** `REPRO` = test kode/kontrak terkontrol; `WS` = koneksi WebSocket lokal nyata; `FS` = operasi pada file penanda audit; `SOURCE` = hasil penelusuran kode yang belum diuji pada layanan aktif.

## Ringkasan pengujian

| Pemeriksaan | Hasil |
|---|---|
| `go test ./... -count=1 -timeout=180s` | Lulus pada 7 package yang memiliki test; package lain tidak memiliki test |
| `go vet ./...` | Lulus, exit 0 |
| Test frontend (seluruh 6 file test yang tersedia) | 37 lulus, 0 gagal |
| `npx tsc --noEmit --incremental false` pada frontend | Lulus, tanpa diagnostik |
| `npm run lint` pada frontend | Exit 0; 0 error, 79 warning |
| Test audit tambahan via Go overlay | **21 skenario: 20 gagal terhadap ekspektasi yang benar, 1 kontrol lulus** |
| Database/container/kamera/perangkat mobile | Tidak diuji langsung; keterbatasan di atas berlaku |

Test audit sengaja menyatakan batas perilaku yang seharusnya dipenuhi. Kegagalannya adalah bukti reproduksi masalah, bukan perbaikan atau kegagalan suite bawaan. Beberapa test menunjuk temuan yang sama; jumlah test bukan jumlah temuan. Sumber test disimpan sebagai `.go.txt` dan hanya dimasukkan saat `go test -overlay`, sehingga tidak menambah kode yang dikompilasi ke aplikasi.

## High

### H01 — Jalur video melewati autentikasi aplikasi

- **Bukti: SOURCE.** `nginx/nginx.conf:97` dan `:111` meneruskan `/stream/` serta `/hls/` langsung ke MediaMTX. `mediamtx.yml:41` memberi `user: any`, password kosong, dan izin `read`/`playback` untuk seluruh path. Tidak ada pemeriksaan sesi/role pada jalur proxy tersebut.
- **Pemicu:** pemanggil mengetahui path kamera yang terdaftar dan mengakses jalur video langsung. Perlindungan admin pada `/api/streams/*` tidak ikut memeriksa request ini.
- **Dampak:** dengan konfigurasi repo, pemanggil tanpa sesi admin dapat mencoba membaca video; mengetahui UUID path menjadi satu-satunya penghalang praktis. Ketersediaan video dan penerapan konfigurasi ini di lingkungan aktif belum diuji.
- **Saran:** validasi sesi admin pada jalur WHEP/HLS atau gunakan otorisasi/token pembacaan MediaMTX yang terikat pengguna dan kamera; uji juga pemutusan akses setelah logout.

### H02 — Isolasi site berlaku pada daftar, tetapi lolos pada akses objek

- **Bukti: REPRO.** `GetCCTVByID` (`backend/handlers/cctv.go:504`) mencari hanya berdasarkan ID. `CreateMaintenanceLog` (`backend/handlers/maintenance.go:65`) memeriksa keberadaan CCTV tanpa mencocokkan site aktif. Beberapa akses terkait seperti QR, scan, snapshot/timeline/health-history dan detail zona juga tidak memakai pemeriksaan objek terhadap site aktif.
- **Reproduksi:** teknisi site A memperoleh detail CCTV site B dengan HTTP **200**, dan membuat log untuk CCTV site B dengan HTTP **201**. Ekspektasi audit: ditolak. Kontrol `CreateFieldMaintenance` pada kondisi yang sama benar-benar menghasilkan **403**.
- **Dampak:** mengetahui ID atau QR suatu objek dapat membuka data atau penulisan lintas kawasan. Middleware yang memvalidasi site pilihan belum cukup bila handler tidak membandingkan site objek.
- **Saran:** terapkan scope objek yang sama pada list/detail/scan/QR/penulisan dan seluruh turunan CCTV. Untuk pemeliharaan, ownership serta site harus diperiksa bersama; ownership saja tidak menutup akses setelah aset dipindahkan.
- **Test:** `TestAudit_CCTVDetailMustDenyOtherSite`, `TestAudit_MaintenanceCreateMustDenyOtherSite`, dan kontrol `TestAudit_FieldCheckMustDenyOtherSite_Control`.

### H03 — Teknisi dapat menggunakan stok pada tugas orang lain

- **Bukti: REPRO.** Route penggunaan suku cadang mengizinkan sesi teknisi (`backend/routes/spare_part.go:37`). `RecordSparePartUsage` (`backend/handlers/spare_part.go:440`) hanya memeriksa log ada dan berstatus completed; tidak memeriksa pemilik, site log, atau kecocokan site suku cadang. Handler baca `:584` juga tidak memeriksa akses ke log.
- **Reproduksi:** pengguna teknisi yang berbeda dari pemilik log tetap mendapat **201**, termasuk eksekusi pengurangan stok dan pembuatan usage di mock. Pembacaan usage log lain mendapat **200**.
- **Dampak:** stok bisa berkurang atas pekerjaan yang bukan tanggung jawab pengguna, serta rincian inventaris/tugas lain dapat terbaca.
- **Saran:** verifikasi ownership/site pada GET dan POST penggunaan; validasi stok yang dipilih memang boleh dipakai untuk lokasi tugas tersebut.
- **Test:** `TestAudit_SparePartWriteMustDenyOtherTechnician` dan `TestAudit_SparePartReadMustDenyOtherTechnician`.

### H04 — Mengganti lampiran bisa menghapus file di luar folder unggahan

- **Bukti: FS + REPRO.** `CreateMaintenanceLogRequest.ImageUrl` tidak dibatasi path. `UpdateMaintenanceLog` (`backend/handlers/maintenance.go:385`) membuat `oldPath := "." + logEntry.ImageUrl` kemudian memanggil `os.Remove` tanpa memverifikasi direktori. Penghapusan dilakukan sebelum penyimpanan database berhasil.
- **Reproduksi:** record fixture memiliki URL yang menunjuk file penanda audit di luar uploads. Saat lampiran diganti, handler merespons **200** dan file penanda tersebut hilang. File lain tidak dijadikan target.
- **Dampak:** pengguna yang bisa menulis lampiran dan kemudian memperbarui log dapat menarget file yang dapat dihapus oleh proses aplikasi. Batas filesystem/read-only pada deployment dapat membatasi target, tetapi tidak menutup kesalahan handler.
- **Saran:** simpan ID unggahan, bukan path bebas; validasi kepemilikan, canonical path dan containment, lalu hapus file lama setelah transaksi sukses dan setelah memastikan tidak dipakai record lain.
- **Test:** `TestAudit_UpdateImageMustNotDeleteOutsideUploads`.

### H05 — Pemakaian stok bersamaan menghasilkan lost update

- **Bukti: REPRO dengan interleaving SQL simulasi.** Transaksi `RecordSparePartUsage` membaca stok tanpa row lock, menghitung `newStok` di memori, lalu menulis angka absolut (`backend/handlers/spare_part.go:483–508`). Transaksi saja tidak melindungi perhitungan dari dua pembacaan nilai lama yang sama.
- **Reproduksi:** dua request sukses, masing-masing memakai 4 dari stok awal 10. Kedua UPDATE memakai nilai **6**, padahal stok akhir semestinya **2**. Dua BEGIN/COMMIT dan dua UPDATE dipastikan terkonsumsi oleh mock. Argumen matcher tercatat beberapa kali karena pencocokan unordered, bukan empat transaksi.
- **Dampak:** riwayat mencatat pemakaian 8 tetapi stok tersisa 6; pola serupa dapat menyetujui total pemakaian melebihi stok.
- **Saran:** gunakan pengurangan atomik dengan kondisi `stok >= jumlah` dan periksa RowsAffected, atau kunci baris selama validasi/pengurangan. Verifikasi ulang pada PostgreSQL saat layanan tersedia.
- **Test:** `TestAudit_ConcurrentStockUsageMustDeductBothRequests`.

### H06 — Event pemeliharaan disiarkan lintas site dan pemilik

- **Bukti: WS.** Create/update/complete pemeliharaan dan generator memanggil `BroadcastEvent` dengan log (`backend/handlers/maintenance.go:137`, `:404`, `:790`; `schedule_runner.go:138`). Implementasi `backend/utils/websocket.go:103` mengirim ke seluruh koneksi tanpa menyeleksi role, site, atau pemilik.
- **Reproduksi:** koneksi teknisi site B benar-benar menerima `MAINTENANCE_UPDATED` berisi catatan site A pada WebSocket lokal. Pengecekan site yang tersedia di `BroadcastToSite` tidak dipakai jalur ini.
- **Dampak:** pembatasan REST list tidak mencegah kebocoran catatan dan identitas pekerjaan lewat koneksi real-time, termasuk antar-teknisi.
- **Saran:** kirim payload minimal ke penerima yang lolos aturan site dan ownership; gunakan jalur terarah secara konsisten.
- **Test:** `TestAudit_MaintenanceBroadcastMustDenyOtherSite`.

### H07 — Tugas dapat dinyatakan selesai memakai checklist karangan

- **Bukti: REPRO.** `CompleteMaintenanceLog` (`backend/handlers/maintenance.go:660–687`) menerima minimal satu item bernama bebas dan memberi completed jika semua item kiriman bernilai ok. Tidak ada pembacaan template aktif/wajib untuk memastikan item yang semestinya diperiksa sudah disertakan. Bukti foto hanya diperiksa prefix URL.
- **Reproduksi:** satu item bernama bukan template resmi dengan hasil ok, ditambah URL `/api/uploads/does-not-exist.jpg`, diterima **200** dan tugas menjadi **completed**. Fixture tidak menyediakan file foto tersebut.
- **Dampak:** hasil pemeriksaan, riwayat pekerjaan, dan metrik selesai dapat menyatakan sukses walaupun pemeriksaan wajib dilewati atau foto tidak ada. Jalur field-maintenance juga mempercayai item dari klien.
- **Saran:** validasi ID template aktif sesuai site di server, kelengkapan item wajib, duplikasi, serta referensi unggahan yang benar-benar ada dan dimiliki pengguna. Template dinamis tetap didukung; jumlah tidak perlu dikunci sembilan.
- **Test:** `TestAudit_CompletionMustRejectInventedChecklist`.

## Medium

### M01 — Waktu dari form pemeliharaan bergeser delapan jam

- **Bukti: REPRO.** Web mengirim `YYYY-MM-DD HH:MM:SS` dari datetime-local (`frontend/src/app/maintenance/page.tsx`). Backend memakai `time.Parse` tanpa lokasi (`backend/handlers/maintenance.go:99` dan `:367`), sehingga hasilnya UTC.
- **Aktual:** input 31 Oktober 2026 pukul 09.00 diterima sebagai `2026-10-31T09:00:00Z`, yaitu **17.00 WITA**, bukan 09.00 WITA. Filter tanggal utilitas juga menggunakan UTC untuk input tanggal tanpa zona.
- **Dampak/saran:** jadwal, tampilan edit, overdue dan batas periode bisa salah; gunakan RFC3339 dengan offset atau parse input lokal memakai zona aplikasi secara konsisten.
- **Test:** `TestAudit_MaintenanceDateMustUseWITA`.

### M02 — Kebijakan SLA tidak dapat dihubungkan ke jadwal lewat API normal

- **Bukti: REPRO + penelusuran field.** Model memiliki `SLAPolicyID`, tetapi request create/update jadwal tidak memiliki field itu (`backend/handlers/maintenance_schedule.go:16`, `:27`). Dashboard/checker SLA hanya mengambil log melalui jadwal yang memiliki `sla_policy_id` (`sla.go:378`, `:572`).
- **Aktual:** POST jadwal dengan `sla_policy_id` valid berakhir **201** tetapi policy tersimpan pada objek sebagai nil. Field JSON diabaikan. Tidak ditemukan setter aplikasi lain pada penelusuran sumber backend non-test.
- **Dampak/saran:** admin dapat membuat master SLA tetapi jadwal normal tidak ikut dimonitor; tambahkan pemilihan/validasi policy dan definisikan aturan untuk pekerjaan manual serta jenis tugas.
- **Test:** `TestAudit_ScheduleMustAcceptSLAPolicy`.

### M03 — Pekerjaan yang belum jatuh tempo menurunkan kepatuhan SLA

- **Bukti: REPRO.** `GetDashboardSLA` menambah `pt.Total` dan `tr.TotalTasks` untuk semua baris (`backend/handlers/sla.go:424`, `:438`), lalu membagi selesai tepat waktu dengan total tersebut. Ini berbeda dari komentar fungsi yang menyebut tugas terminal atau aktif yang sudah melewati deadline sebagai basis evaluasi.
- **Aktual:** satu tugas ongoing yang baru dibuat, target 24 jam ke depan, tetap masuk denominator penilaian, sehingga menghasilkan kepatuhan 0% untuk slot tersebut.
- **Dampak/saran:** dashboard/ranking dapat memburuk hanya karena pekerjaan baru dibuat; pisahkan pending, assessed, compliant dan violated sebelum menghitung persentase.
- **Test:** `TestAudit_SLAComplianceMustExcludeUnfinishedBeforeDeadline`.

### M04 — Zona baru hilang dari daftar site yang sedang dipilih

- **Bukti: REPRO.** `CreateZone` tidak menyalin site aktif dan request tidak menerima SiteID (`backend/handlers/zone.go:195–217`), sedangkan list dan dropdown memfilter `zones.site_id` (`:70`, `:130`). Migrasi hanya mengisi site pada data lama saat migrasi, bukan default insert baru.
- **Aktual:** create pada konteks site terpilih mengembalikan **201** dengan SiteID nil; zona itu kemudian tidak memenuhi filter site yang sama.
- **Dampak/saran:** admin merasa pembuatan gagal dan berpotensi mengulang; tetapkan site pada create dan validasi relasi zona/CCTV pada penugasan.
- **Test:** `TestAudit_ZoneCreateMustPersistSelectedSite`.

### M05 — Kalender mengambil tanggal yang berbeda dari tanggal pengelompokan

- **Bukti: SOURCE.** Kalender mengirim batas bulan ke `/maintenance-logs` (`frontend/src/app/kalender/page.tsx:146`) lalu mengelompokkan hasil berdasarkan scheduled_date (`:161`). Backend memakai parameter itu untuk `created_at` (`backend/handlers/maintenance.go:194`). Kalender juga hanya meminta page 1, limit 100, tanpa mengambil halaman selanjutnya.
- **Pemicu:** tugas dibuat September untuk jadwal Oktober; saat membuka Oktober tugas tidak ikut diambil. Di atas 100 record yang cocok, sisanya juga hilang dari kalender.
- **Dampak/saran:** pekerjaan yang harus dilakukan bisa tidak terlihat; sediakan filter scheduled_date khusus kalender dan ambil seluruh halaman dalam rentang, atau endpoint kalender tersendiri.

### M06 — Filter tanggal laporan pengecekan lapangan diabaikan

- **Bukti: REPRO.** UI laporan kustom menyediakan start/end date untuk field-maintenance, tetapi `fetchFieldMaintenanceReportRows` (`backend/handlers/report_builder.go:444`) hanya memakai overall_status, lalu mengambil semua tanggal.
- **Aktual:** permintaan Oktober 2026 tetap menerima fixture Januari 2020. Query yang dieksekusi memang tidak memuat predikat tanggal.
- **Dampak/saran:** laporan periode dapat berisi data lama dan rekap yang salah; samakan dukungan filter UI dengan query server dan validasi tanggal sebelum ekspor.
- **Test:** `TestAudit_FieldReportMustApplyDateFilter`.

### M07 — Jadwal bulanan bisa melewati bulan; jadwal berulang juga bergeser

- **Bukti: REPRO fungsi.** `nextRunForFrequency` memakai `AddDate(0,1,0)` (`backend/handlers/maintenance_schedule.go:42`); generator menghitung nextRun dari waktu proses aktual (`backend/handlers/schedule_runner.go:113`), bukan waktu rencana sebelumnya.
- **Aktual:** 31 Januari 2027 menjadi **3 Maret 2027**, tanpa kejadian Februari. Tugas yang direncanakan 09.00 tetapi diproses 09.45 menggeser jadwal berikutnya menjadi 09.45.
- **Dampak/saran:** frekuensi dan jam pemeliharaan tidak stabil. Tentukan aturan tanggal akhir bulan secara eksplisit dan gunakan waktu rencana sebagai anchor; kebijakan mengejar kejadian saat server mati juga perlu ditetapkan sebelum perbaikan.
- **Test:** `TestAudit_MonthlyScheduleMustNotSkipFebruary`, `TestAudit_GeneratorMustNotDriftFromPlannedTime`.

### M08 — Penyelesaian tugas belum terlindungi dari request bersamaan

- **Bukti: SOURCE.** Pemeriksaan ownership/status ongoing dilakukan sebelum transaksi (`backend/handlers/maintenance.go:613–642`). Transaksi kemudian membuat field-maintenance dan menyimpan seluruh log tanpa conditional update status/version (`:703–746`). Migrasi menyediakan index biasa, bukan unique, pada `field_maintenances.maintenance_log_id`.
- **Pemicu:** dua request melihat status ongoing yang sama, misalnya double submit, retry jaringan, atau dua tab.
- **Dampak/saran:** keduanya dapat membuat hasil berbeda; penautan terakhir menimpa hasil yang ditunjuk log, sementara hasil lain tetap ada. Gunakan row lock/conditional state transition, idempotency key, dan constraint hasil per tugas sesuai kebutuhan. Belum direproduksi dengan transaksi PostgreSQL nyata.

### M09 — Idempotensi generator jadwal belum aman untuk dua worker

- **Bukti: SOURCE.** `processDueSchedule` melakukan SELECT "sudah ada?", INSERT log, dan SAVE jadwal secara terpisah (`backend/handlers/schedule_runner.go:75–120`). Skema yang ditelusuri tidak memberi unique untuk satu kejadian jadwal. Pemeriksaan juga mengabaikan log soft-deleted.
- **Pemicu:** dua instance/job membaca kejadian yang sama sebelum salah satunya insert, atau proses berhenti di antara insert log dan pembaruan jadwal.
- **Dampak/saran:** duplikasi tugas atau nextRun tidak sinkron; buat claim kejadian dan pembuatan log atomik, gunakan constraint unik pada identitas kejadian, serta lakukan recovery pembaruan jadwal. Belum direproduksi pada dua instance aktif.

### M10 — Audit log memakai ID objek sebagai ID pengguna pelaku

- **Bukti: SOURCE.** Parameter keenam `utils.LogAudit` adalah `userID` pelaku (`backend/utils/audit.go:30`). Beberapa pemanggil mengirim `&logID`, `&zoneID`, atau `&siteID`: `spare_part.go:546`, `zone.go:439`, `:544`, `site.go:521`.
- **Pemicu/dampak:** penggunaan stok atau penugasan zona/site mencoba mencari username berdasarkan ID objek. Dengan FK `audit_logs.user_id -> users.id` pada migrasi awal, insert audit biasanya gagal; tanpa FK, pelaku salah/tercatat system. Operasi bisnis sendiri tetap dapat sukses sehingga jejaknya hilang.
- **Saran:** biarkan actor diambil dari Locals; sediakan resource ID terpisah bila dibutuhkan. Keberadaan constraint pada DB saat ini belum diperiksa.

### M11 — Penugasan teknisi bisa menciptakan tugas yang tidak dapat diakses

- **Bukti: SOURCE.** `AssignMaintenanceLog` memvalidasi user ada dan role teknisi (`backend/handlers/maintenance.go:478–490`), tetapi tidak memvalidasi keanggotaan site CCTV. Create/update jadwal mempunyai pola serupa. Sementara SiteContext menolak teknisi tanpa penugasan site atau pemilihan site di luar user_sites.
- **Pemicu/dampak:** admin memilih teknisi yang bukan anggota site tugas; tugas/notifikasi terbuat, tetapi daftar/detail operasional dapat kosong atau 403 bagi penerima. Pergantian role/penghapusan user juga tidak merekonsiliasi jadwal aktif.
- **Saran:** verifikasi kesesuaian site saat assign/create/update jadwal dan siapkan mekanisme pengalihan ketika membership/role berubah. Jangan diam-diam menambah membership tanpa keputusan bisnis.

### M12 — Wajib ganti password ditegakkan oleh UI, belum oleh API

- **Bukti: REPRO middleware.** UI mengalihkan pengguna ke change-password; `Protected` dan `ValidateTokenVersion` hanya memvalidasi token/version, tidak membatasi pengguna dengan must_change_password (`backend/middleware/auth.go`, `backend/utils/jwt.go:70`).
- **Aktual:** fixture user dengan flag true tetap lolos middleware admin ke endpoint operasional uji dengan **200**.
- **Dampak/saran:** pemanggil API bisa terus memakai sandi sementara tanpa menyelesaikan penggantian; batasi endpoint sesi tersebut ke profil, ganti sandi dan logout sampai flag selesai. Akun tetap harus memiliki kredensial valid; ini bukan bypass login.
- **Test:** `TestAudit_ForcedPasswordMustRestrictOperationalAPI`.

### M13 — Auto-login/refresh mobile membaca bentuk respons yang salah

- **Bukti: REPRO kontrak.** `/auth/me` mengembalikan pengguna langsung dalam `data` (`backend/handlers/auth.go`, GetMyProfile). Mobile membaca `data.user` saat refresh dan bootstrap (`mobile/src/auth/AuthContext.tsx:52–57`, `:83–86`). Login awal memang memakai bentuk data.user yang berbeda.
- **Aktual:** test kontrak terhadap GetMyProfile membuktikan `data.user` tidak ada. Pada mobile, nilai pengguna menjadi undefined; penyimpanan ke secure store juga menerima nilai yang bukan objek pengguna.
- **Dampak/saran:** sesi mobile yang sebelumnya berhasil login dapat kembali dianggap tidak login saat dibuka ulang/refresh; perbaiki kontrak khusus endpoint ini dan uji cold start. Tidak ada emulator/perangkat yang dijalankan.
- **Test:** `TestAudit_MobileProfileContractMustMatchBackend`.

### M14 — Klaim offline-first mobile belum didukung alur penyimpanan/sinkronisasi

- **Bukti: SOURCE.** `uploadPhotos()` dilakukan sebelum blok yang menyimpan draft (`mobile/src/screens/CheckupWizardScreen.tsx:162–177`), sehingga kegagalan upload tidak menyimpan draft. Penelusuran `mobile/src` menemukan getDrafts/removeDraft hanya dipakai di modul storage sendiri, tanpa pengirim ulang antrean. UI tetap menyatakan draft "akan disinkronkan saat online" (`:178`).
- **Pemicu/dampak:** sinyal hilang sebelum upload selesai: checklist/foto tidak disimpan sebagai draft. Bila hanya POST terakhir gagal, draft tersimpan tetapi belum ada pemroses sinkronisasi; referensi foto yang telanjur diunggah juga dapat kedaluwarsa sebagai orphan.
- **Saran:** simpan draft lokal beserta foto lebih dahulu, pisahkan kegagalan validasi dari kegagalan jaringan, lalu tambahkan retry queue dengan idempotensi dan identitas pemilik. Mobile disebut masih in progress pada README utama; ini temuan sumber mobile, bukan klaim insiden deployment.

### M15 — Relasi CCTV pada respons pemeliharaan membocorkan field RTSP mentah

- **Pembaruan database 23.00 WITA:** sebelumnya H08 (High), kini Medium. Semua 35 URL CCTV existing memakai prefix enkripsi enc:v1:; tidak ditemukan URL plaintext. Cacat respons tetap ada, tetapi dampak plaintext pada data saat ini tidak terbukti.
- **Bukti: REPRO + pemeriksaan database read-only.** Model CCTV menyerialisasi `rtsp_url` dan `rtsp_password`. Endpoint CCTV utama memanggil sanitizer, tetapi preload CCTV pada pemeliharaan dikirim langsung (`backend/handlers/maintenance.go:165`; `backend/models/cctv.go`; `cctv.go:64`).
- **Reproduksi:** respons list untuk teknisi memuat RTSP URL fixture dengan password uji dan field password terenkripsi. Nilai ini tidak dihapus oleh sanitizer.
- **Dampak:** bila ada URL legacy plaintext yang masih didukung kode, kredensialnya terbaca; bila seluruh data sudah terenkripsi, ciphertext tetap bocor. Audit ini tidak menyatakan bahwa database saat ini berisi kredensial plaintext.
- **Saran:** gunakan DTO CCTV yang aman untuk setiap nested response dan payload event; sembunyikan field rahasia pada model JSON sebagai pertahanan tambahan.
- **Test:** `TestAudit_MaintenanceListMustRedactNestedRTSP`.

## Low

### L01 — Tanggal update tidak valid dibalas sukses

- **Bukti: REPRO.** `UpdateMaintenanceLog` hanya mengubah scheduled_date bila parsing berhasil (`backend/handlers/maintenance.go:367–382`), tetapi tidak mengembalikan error bila semua format gagal.
- **Aktual/dampak:** `scheduled_date: "not-a-date"` mendapat **200**; nilai lama tetap dipakai. Pengguna mengira perubahan tersimpan.
- **Saran/test:** kembalikan error validasi per field; `TestAudit_InvalidScheduledDateMustBeRejected`.

### L02 — Koordinat NaN diterima sebagai lokasi valid

- **Bukti: REPRO fungsi.** `validateCoordinates` memakai ParseFloat dan pemeriksaan batas tanpa menolak NaN (`backend/handlers/field_maintenance.go:100–122`). Perbandingan `<` dan `>` terhadap NaN keduanya false.
- **Aktual/dampak:** string `NaN` untuk kedua koordinat diterima; peta/rekap lokasi dapat menerima koordinat yang tidak bisa dipakai.
- **Saran/test:** tolak nilai non-finite sebelum validasi rentang; `TestAudit_FieldCoordinatesMustRejectNaN`.

### L03 — Label site default tidak selalu mencerminkan scope data

- **Bukti: SOURCE.** Tanpa cookie, SiteSwitcher menampilkan site default (`frontend/src/components/SiteSwitcher.tsx:67–69`), sedangkan backend sengaja memberi admin scope global saat site tidak dikirim (`backend/middleware/site.go:43`). Menu hanya berisi site individual, tidak menyediakan pilihan untuk menghapus cookie dan kembali ke semua site.
- **Dampak/saran:** admin dapat melihat label satu kawasan padahal data global, dan sulit kembali ke global setelah memilih kawasan. Tampilkan label Semua Site pada scope global dan sediakan pilihan eksplisit untuk kembali ke sana.

## Batas dan hal yang belum dijadikan temuan pasti

- Audit menyatakan adanya cacat pada kode yang ditinjau, bukan bahwa semua potensi bug sudah ditemukan atau seluruh fitur sudah lolos penerimaan. Hasil suite bawaan yang lulus tidak menutup kasus baru di atas.
- Alur media publik H01 dan race PostgreSQL M08/M09 memerlukan verifikasi layanan aktif setelah tersedia. H05 membuktikan urutan read/absolute-write pada mock; belum mengukur perilaku contention database aktif.
- Tidak ada pengujian UI/browser, benchmark, penetrasi layanan publik, pemanggilan kamera/NVR, pengiriman email/webhook ke pihak lain, atau pengujian menggunakan akun nyata.
- Terdapat ketidakkonsistenan dokumen kebijakan GSM pasif dengan verifikasi stream aktif yang baru: kode verifier bisa memicu HLS pada seluruh stream-enabled. Karena dokumentasi terbaru memang menjelaskan verifikasi aktif, hal ini dicatat sebagai **keputusan kebijakan yang perlu diselaraskan**, bukan temuan bug tambahan yang diasumsikan tanpa SOP.
- Efisiensi API key perlu ditinjau pada perbaikan berikutnya: kode memuat kandidat key aktif dan membandingkan bcrypt berurutan. Ini observasi kompleksitas kode; tidak ada angka performa atau klaim bottleneck tanpa pengukuran.
- Lint warning bukan 79 bug produk. Warning tidak dimasukkan satu per satu ke daftar prioritas.

## Artefak bukti dan menjalankan ulang

- `audit-tests.log`: keluaran 20 skenario handler/middleware/kontrak; 19 gagal, 1 kontrol lulus.
- `audit-websocket.log`: satu skenario WebSocket; gagal karena event lintas site diterima.
- `test-results.json`: status seluruh 21 skenario audit.
- `audit_regression_test.go.txt` dan `audit_websocket_test.go.txt`: test audit yang tidak masuk build aplikasi.
- `overlay.json`: pemetaan test virtual untuk checkout pada mesin ini.
- `VERIFIKASI.md`: baseline yang dijalankan, matriks role, dan cara rerun.

**Rekomendasi urutan tindak lanjut:** H01/H04 untuk batas akses video dan file, H02/H03/H06/M15 untuk isolasi dan data, H05/H07 untuk integritas stok serta hasil pekerjaan, lalu sambungan SLA/kalender/jadwal. Audit ini berhenti pada laporan; seluruh perubahan aplikasi menunggu pekerjaan perbaikan terpisah.

## Pembaruan setelah Docker tersedia

Pemeriksaan database nyata selesai 31 Oktober 2026 pukul 23.00 WITA. Lihat [VERIFIKASI-DATABASE.md](VERIFIKASI-DATABASE.md) untuk hasil, perubahan prioritas H08 menjadi M15, kondisi migrasi, dan bukti integritas. Pernyataan Docker/database belum tersedia pada bagian awal laporan adalah catatan tahap pertama; database kini sudah diperiksa read-only. API, kamera, dan layanan aplikasi tetap belum diuji langsung.
