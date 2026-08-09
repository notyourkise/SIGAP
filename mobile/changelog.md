# Changelog - Mobile SIGAP

## [1.0.0] - 2026-08-09

### Fitur: Mobile App (Task 12.4 - Sprint 30)

perubahan yang dilakukan: **Membuat aplikasi mobile SIGAP untuk teknisi di lapangan (React Native / Expo).**

before: Belum ada aplikasi mobile; teknisi harus membawa laptop untuk pengecekan CCTV, scan QR, dan mencatat pemeliharaan.

after:
- Inisialisasi proyek Expo React Native TypeScript di `mobile/` (Android prioritas).
- `react-navigation` (Native Stack + Bottom Tabs): tab Scan, Tugas, Peta, Profil; stack Login, Checkup, Stream, ChangePassword.
- Shared API client (`src/api/client.ts`, axios): mendukung `Authorization: Bearer <token>` (token sesi ditangkap dari header `Set-Cookie` saat login dan disimpan di `expo-secure-store`), helper GET/POST/PUT/DELETE, `extractError`.
- Auth (`src/auth/`): AuthContext (login, auto-login via secure-store + `/auth/me`, refresh, logout), secure-store helper.
- Layar: Login, Profil (ganti password via `/auth/change-password`, status koneksi `/health`, logout), ChangePassword.
- Scan QR (`expo-camera`): pindai QR label CCTV → `POST /api/cctv/scan` → tampilkan detail CCTV.
- Checkup Wizard 9 tahap (`expo-location` GPS + `expo-image-picker` foto): unggah foto via `POST /api/upload`, kirim `POST /api/field-maintenance`; draft offline ke AsyncStorage bila sinyal hilang.
- Peta GIS (`react-native-maps`): marker CCTV + warna status dari `GET /api/cctv`.
- Live View (`react-native-webview`): WHEP fallback untuk monitoring singkat dari `GET /api/streams/:id/url`.
- List Tugas (`/dashboard/tasks`): daftar tugas scheduled/ongoing + petunjuk arah.
- `mobile/README.md` & `mobile/.env.example`.

timestamp: 2026-08-09 WITA
kategori: Besar, Mobile
file yang diubah:
- mobile/package.json (baru)
- mobile/tsconfig.json (baru)
- mobile/app.json (baru)
- mobile/babel.config.js (baru)
- mobile/index.js (baru)
- mobile/App.tsx (baru)
- mobile/src/** (baru)
- mobile/README.md (baru)
- mobile/.env.example (baru)
- mobile/changelog.md (baru)
alasan perubahan: Kebutuhan operasional lapangan (ROADMAP Task 12.4) — memungkinkan teknisi melakukan pengecekan CCTV, scan QR, dan mencatat pemeliharaan dari perangkat mobile tanpa laptop.
