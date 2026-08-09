# SIGAP Mobile (Task 12.4 — React Native / Expo)

Aplikasi mobile untuk **teknisi di lapangan** agar dapat melakukan pengecekan CCTV,
memindai QR label fisik kamera, dan mencatat pemeliharaan tanpa harus membawa laptop.

Target OS: **Android (prioritas utama)**.

## Fitur

- **Auth**: Login (username/password), simpan token sesi aman via `expo-secure-store`,
  auto-login, profil teknisi (ganti password, status koneksi, logout).
- **Scan QR**: Pindai QR label fisik CCTV via kamera, kirim `POST /api/cctv/scan`,
  tampilkan detail CCTV hasil scan.
- **Checkup Wizard**: Checklist pengecekan 9 tahap, ambil GPS otomatis
  (`expo-location`), ambil foto bukti via kamera, unggah + kirim
  `POST /api/field-maintenance`. Mode offline-first menyimpan draft ke
  AsyncStorage saat sinyal hilang.
- **Peta GIS**: Tampilkan sebaran CCTV dengan `react-native-maps` (marker + warna status).
- **Live View**: Pemutar streaming via WebView fallback (WHEP) untuk monitoring singkat.
- **List Tugas**: Daftar tugas `scheduled` / `ongoing` milik teknisi dengan petunjuk arah.

## Struktur

```
mobile/
  App.tsx                      # Root: SafeAreaProvider + AuthProvider + Navigator
  src/
    config.ts                  # API base URL
    api/                       # shared API client (axios) + tipe entitas
    auth/                      # AuthContext + secure-store
    navigation/                # Stack + Bottom Tabs
    screens/                   # Login, Profil, Scan, Checkup, Peta, Stream, Tugas
    components/                # UI reusable
    offline/                   # Draft AsyncStorage
    theme/                     # Warna design system SIGAP
```

## Menjalankan

```bash
cd mobile
npm install
# Atur API URL (lihat .env.example)
cp .env.example .env.local
npm start        # Expo Dev Server
npm run android  # Buka di emulator Android
```

## Verifikasi

```bash
npx tsc --noEmit   # Type-check (harus lulus, 0 error)
```

## Autentikasi

Backend SIGAP memakai sesi cookie httpOnly (`sigap_session`) + token JWT. Untuk
klien non-browser, aplikasi menangkap token sesi dari header `Set-Cookie` saat
login, menyimpannya di `expo-secure-store`, lalu mengirimkannya ulang sebagai
`Authorization: Bearer <token>` (pola fallback Bearer yang didukung backend).
Request mutasi via Bearer dilewati middleware CSRF (pola double-submit).

Detail API: `docs/ROADMAP.md` Fase 12 Task 12.4.
