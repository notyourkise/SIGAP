import axios, { AxiosError, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios';
import { API_BASE_URL, SESSION_COOKIE } from '../config';
import { getSessionToken, storeSessionToken } from '../auth/secureStore';

// Standard response wrapper backend: { status, data, message, errors }.
export interface ApiResponse<T = unknown> {
  status: 'success' | 'error';
  data?: T;
  message?: string;
  errors?: Array<{ field: string; tag: string; value?: string }>;
}

// Client axios bersama. Seluruh request menyertakan Authorization Bearer token
// (cookie httpOnly tidak otomatis dikirim klien non-browser). Menangkap token
// sesi dari Set-Cookie saat login.
export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 20000,
  headers: { 'Content-Type': 'application/json' },
});

// Suntikkan token sesi dari secure-store ke setiap request.
apiClient.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    const token = await getSessionToken();
    if (token) {
      config.headers = config.headers ?? {};
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Menangkap token sesi dari header Set-Cookie pada respons login, lalu
// menyimpannya ke secure-store agar request berikutnya terautentikasi.
apiClient.interceptors.response.use(
  (response) => {
    const setCookie = response.headers['set-cookie'];
    if (setCookie) {
      const session = findCookie(setCookie, SESSION_COOKIE);
      if (session) {
        void storeSessionToken(session);
      }
    }
    return response;
  },
  (error) => Promise.reject(error)
);

// Helper untuk membaca nilai cookie dari array header Set-Cookie.
function findCookie(setCookie: string[], name: string): string | null {
  for (const raw of setCookie) {
    const parts = raw.split(';');
    const first = parts[0]?.trim() ?? '';
    if (first.startsWith(`${name}=`)) {
      return first.slice(name.length + 1);
    }
  }
  return null;
}

// Membungkus error axios menjadi pesan yang ramah.
export function extractError(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ApiResponse>;
    if (axiosError.response?.data?.message) {
      return axiosError.response.data.message;
    }
    if (axiosError.code === 'ECONNABORTED') {
      return 'Waktu permintaan habis. Periksa koneksi Anda.';
    }
    if (!axiosError.response) {
      return 'Tidak dapat terhubung ke server. Periksa koneksi internet.';
    }
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'Terjadi kesalahan yang tidak diketahui.';
}

/** Helper request GET standar. */
export async function apiGet<T>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
  const res = await apiClient.get<ApiResponse<T>>(url, config);
  return res.data;
}

/** Helper request POST standar. */
export async function apiPost<T>(
  url: string,
  body?: unknown,
  config?: AxiosRequestConfig
): Promise<ApiResponse<T>> {
  const res = await apiClient.post<ApiResponse<T>>(url, body, config);
  return res.data;
}

/** Helper request PUT standar. */
export async function apiPut<T>(
  url: string,
  body?: unknown,
  config?: AxiosRequestConfig
): Promise<ApiResponse<T>> {
  const res = await apiClient.put<ApiResponse<T>>(url, body, config);
  return res.data;
}

/** Helper request DELETE standar. */
export async function apiDelete<T>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
  const res = await apiClient.delete<ApiResponse<T>>(url, config);
  return res.data;
}
