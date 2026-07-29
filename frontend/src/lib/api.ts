export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000/api';

export interface ApiResponse<T> {
  status: string;
  message?: string;
  data?: T;
}

export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('sigap_token') : null;

  const headers = new Headers(options.headers);
  if (!(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const config: RequestInit = {
    ...options,
    headers,
  };

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${endpoint}`, config);
  } catch {
    throw new Error(
      'Tidak dapat terhubung ke server SIGAP. Pastikan backend berjalan di localhost:3000.'
    );
  }

  if (response.status === 401) {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('sigap_token');
      localStorage.removeItem('sigap_user');
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
    }
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
  const token = localStorage.getItem('sigap_token');
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });

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
