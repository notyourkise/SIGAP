// Konfigurasi global aplikasi mobile.
// API_BASE_URL menunjuk ke backend SIGAP. Ubah sesuai environment.
// Pada emulator Android gunakan http://10.0.2.2:3000 untuk localhost host.
export const API_BASE_URL = process.env.EXPO_PUBLIC_API_URL || 'http://10.0.2.2:3000/api';

// Nama cookie sesi (konsisten dengan backend utils/session.go).
export const SESSION_COOKIE = 'sigap_session';
export const CSRF_COOKIE = 'sigap_csrf';
