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
//
// Status terverifikasi (stream_verified_state) berasal dari verifikasi aktif
// end-to-end lewat MediaMTX. Nilai ini dipakai bila masih segar, sehingga badge
// tidak berubah menjadi "idle" hanya karena tidak ada penonton.

export type StreamState = 'live' | 'idle' | 'offline' | 'unknown';

export interface LiveBadgeInput {
  stream_live?: boolean | null;
  stream_state?: StreamState | null;
  stream_last_active_at?: string | null;
  network_type?: 'metro-e' | 'gsm' | null;
  stream_verified_state?: StreamState | null;
  stream_verified_at?: string | null;
}

export interface LiveBadge {
  color: string;
  label: string;
  dot: string;
}

const GREEN = { color: '#22C55E', label: 'Online (Live)', dot: '🟢' };
const GREEN_VERIFIED = { color: '#16A34A', label: 'Online terverifikasi', dot: '🟢' };
const BLUE = { color: '#3B82F6', label: 'Siaga (on-demand)', dot: '🔵' };
const YELLOW = { color: '#EAB308', label: 'Pernah aktif hari ini', dot: '🟡' };
const RED = { color: '#EF4444', label: 'Offline', dot: '🔴' };
const RED_VERIFIED = { color: '#DC2626', label: 'Offline terkonfirmasi', dot: '🔴' };
const GRAY = { color: '#9CA3AF', label: 'Tidak diketahui', dot: '⚪' };

// Verifikasi periodik default 15 menit. TTL 30 menit memberi toleransi satu
// siklus terlewat sebelum status terverifikasi dianggap kedaluwarsa.
const VERIFIED_TTL_MS = 30 * 60 * 1000;

export const hasFreshVerification = (cctv: LiveBadgeInput): boolean => {
  if (!cctv.stream_verified_at) return false;
  if (cctv.stream_verified_state !== 'live' && cctv.stream_verified_state !== 'offline') return false;
  const verifiedAt = new Date(cctv.stream_verified_at);
  if (Number.isNaN(verifiedAt.getTime())) return false;
  const age = Date.now() - verifiedAt.getTime();
  return age >= 0 && age <= VERIFIED_TTL_MS;
};

// Status efektif untuk statistik/ringkasan. Status terverifikasi yang masih
// segar menang atas status pasif; selain itu gunakan stream_state apa adanya.
export const getEffectiveStreamState = (cctv: LiveBadgeInput): StreamState => {
  if (hasFreshVerification(cctv)) {
    return cctv.stream_verified_state as StreamState;
  }
  if (cctv.stream_state) return cctv.stream_state;
  if (cctv.stream_live === true) return 'live';
  // Data lama Metro-E memakai probe biner; false berarti offline terkonfirmasi.
  if (cctv.stream_live === false && (cctv.network_type === 'metro-e' || !cctv.network_type)) return 'offline';
  return 'unknown';
};

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

  // Hasil verifikasi aktif yang masih segar adalah bukti terkuat.
  if (hasFreshVerification(cctv)) {
    return cctv.stream_verified_state === 'live' ? GREEN_VERIFIED : RED_VERIFIED;
  }

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
