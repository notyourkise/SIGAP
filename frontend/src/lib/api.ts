export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api';

export interface ApiResponse<T> {
  status: string;
  message?: string;
  data?: T;
}

const SESSION_COOKIE = 'sigap_session';
const CSRF_COOKIE = 'sigap_csrf';
const CSRF_HEADER = 'X-CSRF-Token';

// Baca nilai sebuah cookie (pola double-submit CSRF).
export function getCookie(name: string): string | null {
  if (typeof document === 'undefined') return null;
  const match = document.cookie.match(new RegExp('(?:^|;\\s*)' + name + '=([^;]*)'));
  return match ? decodeURIComponent(match[1]) : null;
}

// Hapus cookie CSRF di sisi klien (dipanggil saat sesi berakhir/401).
// PENTING: Cookie sesi (sigap_session) bersifat httpOnly dan TIDAK BISA
// dihapus oleh JavaScript. Hanya server yang bisa menghapusnya melalui
// header Set-Cookie. Jika JS mencoba menghapusnya, browser justru membuat
// cookie KEDUA (non-httpOnly) dengan nama sama yang berkonflik.
export function clearAuthCookies(): void {
  if (typeof document === 'undefined') return;
  const secure = window.location.protocol === 'https:' ? '; Secure' : '';
  document.cookie = `${CSRF_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax${secure}`;
}

function isMutatingMethod(method?: string): boolean {
  const m = (method || 'GET').toUpperCase();
  return m === 'POST' || m === 'PUT' || m === 'PATCH' || m === 'DELETE';
}

// Penanganan 401: bersihkan cookie, beri tahu AuthContext, dan alihkan ke login.
function handleUnauthorized(): void {
  if (typeof window === 'undefined') return;
  clearAuthCookies();
  window.dispatchEvent(new Event('sigap:auth-expired'));
  if (window.location.pathname !== '/login') {
    window.location.href = '/login';
  }
}

// Opsi tambahan di luar RequestInit standar.
export interface ApiRequestOptions extends RequestInit {
  // Jika true, respons 401 TIDAK akan memicu handleUnauthorized().
  // Gunakan untuk panggilan yang wajar mengembalikan 401 tanpa berarti
  // sesi expired, misalnya bootstrap (/auth/me) dan login (/auth/login).
  skipAuthRedirect?: boolean;
}

export async function apiRequest<T>(
  endpoint: string,
  options: ApiRequestOptions = {}
): Promise<ApiResponse<T>> {
  const { skipAuthRedirect, ...fetchOptions } = options;

  const headers = new Headers(fetchOptions.headers);
  if (!(fetchOptions.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  // Kirim token CSRF untuk permintaan mutasi (dibaca dari cookie non-httpOnly).
  const csrf = getCookie(CSRF_COOKIE);
  if (csrf && isMutatingMethod(fetchOptions.method)) {
    headers.set(CSRF_HEADER, csrf);
  }

  // credentials: 'include' — cookie sesi httpOnly dikirim otomatis.
  const config: RequestInit = {
    ...fetchOptions,
    headers,
    credentials: 'include',
  };

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${endpoint}`, config);
  } catch {
    throw new Error(
      'Tidak dapat terhubung ke server SIGAP. Pastikan backend server berjalan.'
    );
  }

  // Hanya trigger handleUnauthorized untuk 401 yang memang berarti sesi expired.
  // Panggilan bootstrap (/auth/me) dan login (/auth/login) menggunakan
  // skipAuthRedirect=true agar 401 ditangani secara lokal.
  if (response.status === 401 && !skipAuthRedirect) {
    handleUnauthorized();
  }

  let data: ApiResponse<T>;
  try {
    data = await response.json();
  } catch {
    throw new Error(
      response.ok
        ? 'Respons server SIGAP tidak dapat dibaca.'
        : `Server SIGAP mengembalikan kesalahan HTTP ${response.status}.`
    );
  }

  if (!response.ok) {
    throw new Error(data.message || 'Terjadi kesalahan pada server');
  }

  return data;
}

export async function downloadApiFile(endpoint: string, fallbackName: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    credentials: 'include',
  });

  if (response.status === 401) {
    handleUnauthorized();
    throw new Error('Sesi Anda telah berakhir. Silakan masuk kembali.');
  }

  if (!response.ok) {
    let message = 'Gagal mengunduh file';
    try {
      const payload = await response.json();
      message = payload.message || message;
    } catch {
      // Respons unduhan yang gagal tidak selalu berupa JSON.
    }
    throw new Error(message);
  }

  const blob = await response.blob();
  const disposition = response.headers.get('content-disposition');
  const fileName = disposition?.match(/filename="?([^"]+)"?/i)?.[1] || fallbackName;
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = fileName;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}
