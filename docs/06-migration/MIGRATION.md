# MIGRATION — Migrasi Database Versioned (Task 7.1)

> Prosedur migrasi database SIGAP menggunakan **golang-migrate v4** dengan driver
> PostgreSQL (pgx v5). Dokumentasi ini menjelaskan cara kerja, penggunaan CLI,
> penambahan migration baru, serta pemulihan saat migration gagal.
> Diperbarui: 2026-08-06 (Task 7.1, Fase 7 Kesiapan Produksi).

---

## 1. Latar Belakang

Sebelum Task 7.1, skema database dikelola **GORM AutoMigrate** — sinkronisasi
otomatis dari definisi struct di `backend/models/`. Masalahnya:

- Tidak ada riwayat versi skema — sulit menelusuri "skema mana yang sedang berjalan".
- Tidak ada rollback yang andal saat perubahan skema bermasalah.
- Perubahan skema produksi berisiko kehilangan data (AutoMigrate hanya menambah,
  jarang memperbaiki kolom yang sudah ada).

Sejak Task 7.1, skema dikelola lewat **file migration SQL terurut** di
`backend/migrations/`. Fase transisi: AutoMigrate **dikomentari** di `main.go`
(akan dihapus setelah migration stabil), migration file berjalan lebih dulu saat
startup.

## 2. Struktur File Migration

```
backend/migrations/
├── migrations.go                  # embed FS (memuat *.sql ke binary)
├── 000001_init_schema.up.sql      # skema awal: 10 tabel + index + FK
└── 000001_init_schema.down.sql    # rollback skema awal
```

Konvensi penamaan golang-migrate (wajib):

```
<urutan 6 digit>_<nama>.up.sql
<urutan 6 digit>_<nama>.down.sql
```

- Urutan menentukan urutan eksekusi (naik: dari kecil ke besar; turun: terbalik).
- Setiap migration **harus** berpasangan up/down.
- File yang sudah diterapkan **tidak boleh diubah** — buat migration baru
  (mis. `000002_xxx`) jika skema berubah.

## 3. Cara Kerja

1. Saat startup, `config.RunMigrations(db)` (dipanggil dari `main.go` **sebelum**
   operasi lain) menerapkan semua migration yang belum berjalan (`m.Up()`).
2. Riwayat versi dicatat di tabel `schema_migrations` (kolom `version`, `dirty`).
3. Migration dieksekusi via driver pgx v5 langsung pada koneksi GORM yang ada.
4. Idempoten: migration yang sudah tercatat versinya tidak dieksekusi ulang;
   `ErrNoChange` diabaikan dengan aman.

> [!NOTE]
> Seperti kebanyakan database Postgres, migration dieksekusi **tanpa transaksi
> wrapper** (pernyataan dijalankan satu per satu). Bila migration gagal di
> tengah jalan, ikuti langkah pemulihan pada bagian 6.

## 4. CLI Helper (Manual)

Semua perintah dijalankan dari folder `backend/` (env `DB_*` dibaca dari
`backend/.env` seperti aplikasi utama):

```bash
# Terapkan semua migration yang belum berjalan
go run cmd/migrate/main.go up

# Rollback SEMUA migration (kembali ke versi 0) — hati-hati, destruktif
go run cmd/migrate/main.go down

# Tampilkan versi migration aktif + status dirty
go run cmd/migrate/main.go version
```

Contoh keluaran:

```text
🗄️ Migration DB versioned aktif pada versi 1 (dirty=false)
✅ Migration up selesai.
```

## 5. Integrasi CI

`.github/workflows/ci.yml` (job `backend-ci`) menjalankan verifikasi end-to-end
terhadap PostgreSQL nyata (service container):

```bash
go run cmd/migrate/main.go up
go run cmd/migrate/main.go down
go run cmd/migrate/main.go up
go run cmd/migrate/main.go version
```

Setiap perubahan migration file baru diharuskan tetap lulus rangkaian ini.

## 6. Pemecahan Masalah

### 6.1 Status dirty = true

Terjadi bila migration gagal setelah menandai versi (biasanya karena error SQL).
Cek versi dengan `version`, perbaiki penyebab, lalu lakukan salah satu:

- Jika database **belum** dipakai produksi: `DROP TABLE schema_migrations` lalu
  jalankan `up` lagi.
- Jika perlu mundur satu versi: perbaiki migration yang gagal, lalu jalankan
  `up` lagi (golang-migrate akan meneruskan dari versi terakhir yang berhasil).
- Jika migration yang gagal sudah dijalankan sebagian dan tidak bisa diperbaiki:
  periksa manual tabel yang sudah dibuat, sesuaikan, lalu hapus baris versi yang
  salah dari `schema_migrations` (hanya bila Anda yakin skema sudah benar).

### 6.2 Migration pada database eksisting (fase transisi)

Migration `000001` aman dijalankan pada database hasil AutoMigrate lama:

- `CREATE TABLE IF NOT EXISTS` → tabel yang sudah ada dilewati.
- `CREATE INDEX IF NOT EXISTS` / `CREATE UNIQUE INDEX IF NOT EXISTS` → sama.
- Foreign key memakai blok `DO $$ ... IF NOT EXISTS (pg_constraint) ...` karena
  PostgreSQL tidak mendukung `ADD CONSTRAINT IF NOT EXISTS` — constraint yang
  sudah dibuat GORM dengan nama serupa tidak ditimpa.

### 6.3 Kesalahan "no migration"

`version` mengembalikan `0` dan `dirty=false` — belum ada migration yang
diterapkan. Ini normal untuk database kosong; jalankan `up`.

## 7. Menambahkan Migration Baru

Contoh: menambah kolom `serial_number` pada `cctvs`.

1. Buat `backend/migrations/000002_add_serial_number_cctv.up.sql`:

   ```sql
   ALTER TABLE cctvs ADD COLUMN serial_number varchar(50);
   ```

2. Buat `backend/migrations/000002_add_serial_number_cctv.down.sql`:

   ```sql
   ALTER TABLE cctvs DROP COLUMN IF EXISTS serial_number;
   ```

3. Sinkronkan model GORM di `backend/models/cctv.go` (bila perlu) — setelah
   AutoMigrate dihapus, model hanya berfungsi sebagai definisi tipe Go.

4. Uji lokal:

   ```bash
   go run cmd/migrate/main.go up
   go run cmd/migrate/main.go down
   go run cmd/migrate/main.go up
   ```

5. Catat perubahan di `backend/changelog.md`.

> [!IMPORTANT]
> Jangan pernah mengubah file migration yang sudah di-merge ke `main` dan sudah
> dijalankan di environment mana pun. Selalu buat migration baru.

## 8. Referensi

- Implementasi: `backend/config/migrate.go` (`RunMigrations`, `MigrationUp`,
  `MigrationDown`, `MigrationVersion`, `newMigrateInstance`).
- Sumber migration: `backend/migrations/` (embed FS → binary).
- CLI: `backend/cmd/migrate/main.go`.
- Library: `github.com/golang-migrate/migrate/v4` v4.19.1 (driver `database/pgx/v5`,
  sumber `source/iofs`).
- Status task: `docs/ROADMAP.md` Fase 7 (Task 7.1).
