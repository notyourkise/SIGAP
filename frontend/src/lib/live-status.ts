// Logika badge status live CCTV. Dipisahkan dari komponen agar dapat diuji
// tanpa React/DOM dan dipakai bersama oleh peta serta grid streaming.
//
// Empat keadaan stream:
//   live    → sumber MediaMTX siap / probe Metro-E berhasil
//   idle    → path on-demand terdaftar tetapi belum berjalan (tidak ada penonton)
//   offline → kegagalan terkonfirmasi (probe Metro-E gagal / ada penonton tetapi sumber error)
//   unknown → belum dapat disimpulkan (mis. MediaMTX tidak dapat dihubungi)
//
// Kompatibilitas: data lama tanpa stream_state tetap memakai stream_live dan
// stream_last_active_at seperti sebelumnya.

export type StreamState = 'live' | 'idle' | 'offline' | 'unknown';

export interface LiveBadgeInput {
  stream_live?: boolean | null;
  stream_state?: StreamState | null;
  stream_last_active_at?: string | null;
  network_type?: 'metro-e' | 'gsm' | null;
}

export interface LiveBadge {
  color: string;
  label: string;
  dot: string;
}

const GREEN = { color: '#22C55E', label: 'Online (Live)', dot: '🟢' };
const BLUE = { color: '#3B82F6', label: 'Siaga (on-demand)', dot: '🔵' };
const YELLOW = { color: '#EAB308', label: 'Pernah aktif hari ini', dot: '🟡' };
const RED = { color: '#EF4444', label: 'Offline', dot: '🔴' };
const GRAY = { color: '#9CA3AF', label: 'Tidak diketahui', dot: '⚪' };

// Apakah stream_last_active_at berada dalam 24 jam terakhir dan valid.
const recentlyActive = (value?: string | null): boolean => {
  if (!value) return false;
  const lastActive = new Date(value);
  if (Number.isNaN(lastActive.getTime())) return false;
  const hoursAgo = (Date.now() - lastActive.getTime()) / 3_600_000;
  return hoursAgo >= 0 && hoursAgo < 24;
};

export const getLiveBadge = (cctv: LiveBadgeInput): LiveBadge => {
  const { stream_live, stream_state, stream_last_active_at, network_type } = cctv;

  // Status eksplisit dari backend selalu menang.
  switch (stream_state) {
    case 'live':
      return GREEN;
    case 'idle':
      return BLUE;
    case 'offline':
      return RED;
    case 'unknown':
      return network_type === 'gsm' && recentlyActive(stream_last_active_at) ? YELLOW : GRAY;
  }

  // Metro-E: status biner hasil probe langsung.
  if (network_type === 'metro-e') {
    if (stream_live === true) return GREEN;
    if (stream_live === false) return RED;
    return GRAY;
  }

  // GSM tanpa stream_state (data lama): live → hijau; jika tidak, gunakan
  // riwayat aktivitas. Tidak ada bukti offline tanpa kegagalan terkonfirmasi.
  if (network_type === 'gsm') {
    if (stream_live === true) return GREEN;
    return recentlyActive(stream_last_active_at) ? YELLOW : GRAY;
  }

  // Fallback perilaku lama untuk data tanpa network_type.
  if (stream_live === true) return GREEN;
  if (stream_live === false) return RED;
  return GRAY;
};
