# Pedoman AI - Backend (SIGAP)

## Stack Teknologi
- Bahasa: Go (Golang)
- Framework Web: Fiber (gaya penulisan mirip Express.js)
- Database: PostgreSQL (melalui ORM GORM)
- Port Default: Port 3000

## Aturan Penulisan Kode
1. SELALU berikan komentar kode dan penjelasan menggunakan Bahasa Indonesia.
2. Gunakan Layered Architecture. Pisahkan logika ke dalam folder:
   - `models/` untuk definisi struct/skema database dan seeder.
   - `routes/` untuk pendaftaran endpoint API.
   - `handlers/` untuk logika request/response (logika bisnis).
   - `config/` untuk pengaturan koneksi database dan environment.
   - `utils/` untuk helper (seperti enkripsi JWT, validasi, dll.).
3. Selalu kembalikan response API dalam format JSON yang konsisten (contoh: `{ "status": "success", "data": ... }` atau `{ "status": "error", "message": "..." }`).
4. Jangan pernah menambahkan library pihak ketiga kecuali sangat diperlukan dan diminta (misal: JWT v5, bcrypt, driver postgres, gorm).

## 📝 Aturan Pencatatan Changelog (WAJIB)
Setiap perubahan kode (sekecil atau sebanyak apa pun) WAJIB dicatat di `backend/changelog.md` menggunakan format persis berikut:
```text
perubahan yang dilakukan:
before:
after:
timestamp: (Gunakan waktu WITA Asia/Makassar, pada saat perubahan dilakukan)
kategori: berat/sedang/low
file yang diubah:
alasan perubahan:
```

## 🐙 Aturan Git & Alur Kerja GitHub (WAJIB)
1. **DILARANG KERAS** melakukan push langsung (*direct push*) ke branch `main`.
2. Setiap pengerjaan fitur atau perbaikan bug harus dilakukan di dalam **branch baru** yang dibuat terpisah.
3. Setelah pengerjaan di branch baru selesai, developer wajib membuat **Pull Request (PR)**.
4. Setiap Pull Request (PR) **wajib melalui proses review** terlebih dahulu oleh pengembang lain sebelum diperbolehkan untuk di-merge ke branch `main`.

## ⚠️ BATASAN OTORITAS & KEAMANAN KETAT
1. **AI Agent DILARANG KERAS menyentuh, memodifikasi, membuat, membaca, atau menghapus file/folder apa pun di dalam folder `frontend/` tanpa seizin Intan.**
2. Pengerjaan di area backend harus terisolasi sepenuhnya di dalam direktori `backend/`. Jika ada kebutuhan integrasi dengan frontend, komunikasikan terlebih dahulu atau pastikan perubahan frontend telah disetujui oleh Intan secara eksplisit.