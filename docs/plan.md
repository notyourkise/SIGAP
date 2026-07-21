# Dokumen Perencanaan Proyek SIGAP

## 1. Identitas Sistem
- **Nama Proyek:** SIGAP
- **Kepanjangan:** Sistem Identifikasi Gangguan Aset Pusat
- **Tujuan Utama:** Mengelola, memantau, dan mencatat riwayat pemeliharaan aset CCTV kawasan IKN secara terpusat.
- **Arsitektur Repositori:** Monorepo (menggabungkan Frontend dan Backend dalam satu wadah).

## 2. Tech Stack (Tumpukan Teknologi)
- **Backend (API & Logika Server):** Go (Golang) menggunakan framework Fiber (Layered Architecture).
- **Frontend (Antarmuka Pengguna):** Next.js (App Router) menggunakan direktori `src/`.
- **Styling UI:** Tailwind CSS.
- **Database (Rencana):** PostgreSQL.

## 3. Ekosistem & Alat Pendukung
- **AI Code Assistant:** Codex.
- **Version Control:** Git & GitHub.
- **Automation / Task Runner:** `Makefile` di root direktori untuk mengeksekusi perintah terminal secara cepat (misal: menjalankan server BE dan FE secara bersamaan).

## 4. Strategi Dokumentasi & Prompting AI
Proyek ini menggunakan strategi distribusi file dokumentasi untuk memisahkan konteks antara pekerjaan Frontend dan Backend, serta menjaga fokus Codex AI:

- **Aturan AI (`agent.md`):**
  - `/backend/agent.md` -> Berisi aturan spesifik untuk Go Fiber, struktur database, dan logika backend.
  - `/frontend/agent.md` -> Berisi aturan spesifik untuk Next.js, komponen UI, dan Tailwind CSS.
- **Buku Harian Proyek (`CHANGELOG.md`):**
  - `/CHANGELOG.md` (Utama) -> Mencatat pencapaian besar ekosistem monorepo.
  - `/backend/CHANGELOG.md` -> Mencatat pembaruan teknis terkait API dan database.
  - `/frontend/CHANGELOG.md` -> Mencatat pembaruan teknis terkait antarmuka dan komponen UI.

## 5. Peta Struktur Direktori Standar
```text
sigap-monorepo/
├── Makefile
├── CHANGELOG.md
├── README.md
├── plan.md
├── .gitignore
│
├── backend/
│   ├── agent.md
│   ├── CHANGELOG.md
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   ├── config/
│   ├── routes/
│   ├── handlers/
│   └── models/
│
└── frontend/
    ├── agent.md
    ├── CHANGELOG.md
    ├── package.json
    ├── tailwind.config.ts
    └── src/
        ├── app/
        ├── components/
        └── lib/