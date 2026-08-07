export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api';

export interface ApiResponse<T> {
  status: string;
  message?: string;
  data?: T;
  errors?: ApiFieldError[];
}

// Error validasi per field yang dikirim backend (format utils/validator.go):
// { field, tag, value } — mis. { field: "Status", tag: "oneof", value: "aktif pemeliharaan nonaktif" }.
export interface ApiFieldError {
  field: string;
  tag: string;
  value?: string;
}

// Error khusus untuk respons validasi: selain .message yang sudah berisi
// ringkasan, .errors menyimpan detail per field agar halaman bisa
// menampilkan daftar isian yang bermasalah secara terstruktur.
export class ApiValidationError extends Error {
  errors: ApiFieldError[];

  constructor(message: string, errors: ApiFieldError[]) {
    super(message);
    this.name = 'ApiValidationError';
    this.errors = errors;
    Object.setPrototypeOf(this, ApiValidationError.prototype);
  }
}

// Label Bahasa Indonesia per field. Kunci memakai nama JSON (snake_case);
// lookup dilakukan case-insensitive, dengan fallback transformasi nama
// field Go (CamelCase, mis. "IPNVR") menjadi label yang bisa dibaca.
const FIELD_LABELS: Record<string, string> = {
  kode_cctv: 'Kode CCTV',
  id_channel: 'ID Channel',
  nama: 'Nama',
  lokasi_area: 'Lokasi',
  longitude: 'Longitude',
  latitude: 'Latitude',
  jenis_cctv: 'Jenis CCTV',
  sumber_listrik: 'Sumber listrik',
  tipe_koneksi: 'Tipe koneksi',
  indoor_outdoor: 'Pemasangan (indoor/outdoor)',
  pemilik: 'Pemilik / pengelola',
  status: 'Status',
  ip_nvr: 'IP NVR',
  ip_camera: 'IP kamera',
  keterangan: 'Keterangan',
  channel_port: 'Channel port NVR',
  rtsp_password: 'Password RTSP',
  stream_enabled: 'Streaming aktif',
  username: 'Username',
  password: 'Kata sandi',
  old_password: 'Kata sandi lama',
  new_password: 'Kata sandi baru',
  new_password_confirm: 'Konfirmasi kata sandi baru',
  confirm_password: 'Konfirmasi kata sandi',
  full_name: 'Nama lengkap',
  email: 'Email',
  role: 'Peran',
  profile_photo_url: 'Foto profil',
  qr_payload: 'Payload QR',
  cctv_id: 'CCTV',
  cctv_ids: 'CCTV',
  technician_id: 'Teknisi',
  action_taken: 'Tindakan',
  scheduled_date: 'Tanggal terjadwal',
  notes: 'Catatan',
  image_url: 'URL gambar',
  start_date: 'Tanggal mulai',
  end_date: 'Tanggal selesai',
  module: 'Modul',
  search: 'Pencarian',
  checklist_results: 'Hasil checklist',
  photos: 'Foto',
};

function getFieldLabel(field: string): string {
  const key = field.toLowerCase();
  if (FIELD_LABELS[key]) return FIELD_LABELS[key];
  // Transformasi nama field Go (CamelCase) → label terbaca, mis. "IPNVR" → "IPNVR",
  // "NewPassword" → "New Password" — dikapitalisasi huruf pertama.
  const readable = field.replace(/([a-z0-9])([A-Z])/g, '$1 $2');
  return readable.charAt(0).toUpperCase() + readable.slice(1);
}

// Terjemahkan error validasi backend menjadi kalimat per field yang jelas,
// mis. "Status harus berupa salah satu dari: aktif, pemeliharaan, nonaktif".
export function formatValidationMessages(errors: ApiFieldError[]): string[] {
  return errors.map((e) => {
    const label = getFieldLabel(e.field);
    switch (e.tag) {
      case 'required':
        return `${label} wajib diisi`;
      case 'min':
        return `${label} minimal ${e.value || 'N'} karakter`;
      case 'max':
        return `${label} maksimal ${e.value || 'N'} karakter`;
      case 'oneof':
        return `${label} harus berupa salah satu dari: ${(e.value || '').trim().replace(/\s+/g, ', ')}`;
      case 'ip':
        return `${label} harus berupa alamat IP yang valid`;
      case 'ipv4':
        return `${label} harus berupa alamat IPv4 yang valid`;
      case 'email':
        return `${label} harus berupa alamat email yang valid`;
      case 'alpha':
        return `${label} hanya boleh berisi huruf`;
      case 'alphanum':
        return `${label} hanya boleh berisi huruf dan angka`;
      case 'uuid':
      case 'uuid4':
        return `${label} tidak valid`;
      case 'strongpassword':
        return `${label} harus mengandung huruf besar, huruf kecil, angka, dan karakter khusus`;
      case 'datetime':
        return `${label} harus berupa tanggal yang valid`;
      case 'eqfield':
        return `${label} harus sama dengan isian yang bersesuaian`;
      default:
        return `${label} tidak valid (aturan: ${e.tag})`;
    }
  });
}

const SESSION_COOKIE = 'sigap_session';
const CSRF_COOKIE = 'sigap_csrf';
const CSRF_HEADER = 'X-CSRF-Token';

// Baca nilai sebuah cookie (pola double-submit CSRF).
// Ambil kemunculan TERAKHIR: browser bisa menyimpan beberapa cookie dengan
// nama sama namun Path berbeda (stale lama di /api + aktif di /). document.cookie
// diurutkan sesuai RFC 6265 (Path terpanjang lebih dulu), jadi nilai pada path
// terpendek ("/") — cookie sesi/CSRF aktif — selalu berada paling akhir.
export function getCookie(name: string): string | null {
  if (typeof document === 'undefined') return null;
  const regex = new RegExp('(?:^|;\\s*)' + name + '=([^;]*)', 'g');
  let match: RegExpExecArray | null;
  let last: string | null = null;
  while ((match = regex.exec(document.cookie)) !== null) {
    last = decodeURIComponent(match[1]);
  }
  return last;
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
    // Respons validasi backend memuat detail per field (errors[]) yang selama
    // ini dibuang — sekarang diubah menjadi pesan yang menjelaskan apa yang salah.
    if (Array.isArray(data.errors) && data.errors.length > 0) {
      const details = formatValidationMessages(data.errors as ApiFieldError[]);
      throw new ApiValidationError(
        `${data.message || 'Validasi input gagal'}: ${details.join('; ')}`,
        data.errors as ApiFieldError[]
      );
    }
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
