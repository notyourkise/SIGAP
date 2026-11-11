// Tipe entitas yang dikonsumsi dari API backend SIGAP.

export interface User {
  id: string;
  username: string;
  full_name: string;
  email: string;
  role: 'admin' | 'teknisi';
  must_change_password?: boolean;
}

export interface LoginResult {
  csrf_token?: string;
  user: User;
}

export interface CCTV {
  id: string;
  kode_cctv: string;
  id_channel?: string;
  nama: string;
  lokasi_area?: string;
  longitude?: string;
  latitude?: string;
  jenis_cctv?: string;
  status: string;
  health_score?: number;
  ip_camera?: string;
  ip_nvr?: string;
  channel_port?: number;
  stream_enabled?: boolean;
  zone_id?: string | null;
}

export interface Pagination {
  page: number;
  limit: number;
  total_rows: number;
  total_pages: number;
}

export interface CCTVListData {
  rows: CCTV[];
  pagination: Pagination;
}

export interface FieldCheckItem {
  name: string;
  result: 'ok' | 'not_ok';
  note: string;
}

export interface FieldImageItem {
  image_url: string;
}

export interface CreateFieldMaintenancePayload {
  cctv_id: string;
  notes: string;
  check_latitude: string;
  check_longitude: string;
  client_request_id?: string;
  checks: FieldCheckItem[];
  images: FieldImageItem[];
}

export interface UploadResult {
  url: string;
  filename: string;
  size: number;
  type: string;
}

export interface TaskItem {
  id: string;
  cctv_id: string;
  cctv_name: string;
  cctv_kode: string;
  cctv_location: string;
  status: string;
  action_taken: string;
  scheduled_date: string;
  days_overdue: number;
}

export interface TasksData {
  tasks: TaskItem[];
  pending_count: number;
  overdue_count: number;
}

export interface StreamURLData {
  whep_url: string;
  hls_url: string;
  stream_active: boolean;
  stream_ready: boolean;
  viewers: number;
}
