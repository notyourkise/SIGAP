# Pedoman AI - Frontend (SIGAP)

## Stack Teknologi
- Framework: Next.js (App Router)
- Bahasa: TypeScript
- Styling: Tailwind CSS v4 (Tanpa file `tailwind.config`, konfigurasi diletakkan langsung di dalam `globals.css` menggunakan directive `@theme`)
- Struktur: Menggunakan direktori `src/`
- Port Dev: Selalu berjalan di **port 3001** (`npm run dev` dikonfigurasi ke port 3001)

## Aturan Penulisan Kode
1. SELALU berikan komentar kode menggunakan Bahasa Indonesia.
2. Buat komponen UI yang modular dan dapat digunakan ulang (reusable) di dalam folder `src/components/`.
3. Hindari penggunaan CSS kustom, maksimalkan penggunaan utility classes dari Tailwind CSS. Gunakan CSS Variables di `globals.css` hanya untuk keperluan tema/dark mode.
4. Gunakan pustaka ikon `lucide-react` yang sudah terpasang untuk semua kebutuhan ikon grafis.
5. Jangan pernah mencampur adukkan logika fetching data (API call) langsung di komponen UI, gunakan API helper yang sudah disediakan di `src/lib/api.ts` atau tempatkan di `src/lib/`.

## 📝 Aturan Pencatatan Changelog (WAJIB)
Setiap perubahan kode (sekecil atau sebanyak apa pun) WAJIB dicatat di `frontend/changelog.md` menggunakan format persis berikut:
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
1. **AI Agent DILARANG KERAS menyentuh, memodifikasi, membuat, membaca, atau menghapus file/folder apa pun di dalam folder `backend/` tanpa seizin Haikal.**
2. Pengerjaan di area frontend harus terisolasi sepenuhnya di dalam direktori `frontend/`. Jika ada kebutuhan integrasi dengan backend, komunikasikan terlebih dahulu atau pastikan perubahan backend telah disetujui oleh Haikal secara eksplisit.