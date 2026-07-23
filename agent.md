# Aturan Pengembangan Monorepo SIGAP (Untuk Seluruh Agent AI)

## 🚨 ATURAN UTAMA GIT & PUSH (HARAM LANGSUNG KE MAIN)
* **JANGAN PERNAH** melakukan `git push` langsung ke branch `main`. Hukumnya adalah **HARAM**.
* Seluruh pengerjaan fitur baru, perbaikan kode, atau integrasi wajib dilakukan pada **branch baru** yang spesifik dan relevan dengan nama fitur yang sedang dibuat (misalnya `feature/db-pooling`, `feature/export-csv`, dll.).
* Proses penggabungan kode ke `main` **wajib** melalui pembuatan Pull Request (PR) di GitHub.
* Biarkan pipa CI/CD GitHub Actions memverifikasi keaktifan unit testing dan build hingga berstatus sukses (centang hijau) sebelum PR digabungkan ke `main`.

---

## 📝 Aturan Pencatatan Log Perubahan (Changelog)
* Setiap perubahan wajib ditulis di `backend/changelog.md` pada folder backend dengan format:
  ```text
  perubahan yang dilakukan:
  before:
  after:
  timestamp: (WITA Asia/Makassar)
  kategori: berat/sedang/low
  file yang diubah:
  alasan perubahan:
  ```
