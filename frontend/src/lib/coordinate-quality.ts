export type CoordinateQualityStatus =
  | 'verified'
  | 'estimated'
  | 'missing'
  | 'partial'
  | 'invalid'
  | 'zero'
  | 'duplicate';

export type CoordinateQualityFilterKey =
  | 'needs_attention'
  | 'estimated'
  | 'missing_invalid'
  | 'duplicate'
  | 'all';

export interface CoordinateQualityItem {
  id: string;
  kode_cctv: string;
  nama: string;
  lokasi_area: string;
  site_id?: string | null;
  site_nama?: string;
  status: string;
  latitude: number | null;
  longitude: number | null;
  keterangan: string;
  coordinate_status: CoordinateQualityStatus | string;
  issues: string[];
}

export interface CoordinateQualitySummary {
  total_count: number;
  verified_count: number;
  needs_attention_count: number;
  missing_count: number;
  partial_count: number;
  invalid_count: number;
  zero_count: number;
  estimated_count: number;
  duplicate_count: number;
  completeness_percent: number;
  verified_percent: number;
  pending_legacy_codes: string[];
}

export interface CoordinateQualityReport {
  summary: CoordinateQualitySummary;
  items: CoordinateQualityItem[];
}

const MISSING_OR_INVALID_ISSUES = new Set([
  'missing_coordinate',
  'partial_coordinate',
  'invalid_coordinate',
  'zero_coordinate',
]);

export function filterCoordinateQualityItems(
  items: CoordinateQualityItem[],
  filter: CoordinateQualityFilterKey
): CoordinateQualityItem[] {
  switch (filter) {
    case 'needs_attention':
      return items.filter((item) => item.issues.length > 0);
    case 'estimated':
      return items.filter(
        (item) =>
          item.coordinate_status === 'estimated' ||
          item.issues.includes('estimated_coordinate')
      );
    case 'missing_invalid':
      return items.filter((item) =>
        item.issues.some((issue) => MISSING_OR_INVALID_ISSUES.has(issue))
      );
    case 'duplicate':
      return items.filter(
        (item) =>
          item.coordinate_status === 'duplicate' ||
          item.issues.includes('duplicate_coordinate')
      );
    case 'all':
    default:
      return items;
  }
}

export function coordinateStatusLabel(status: string): string {
  switch (status) {
    case 'verified':
      return 'Terverifikasi';
    case 'estimated':
      return 'Estimasi';
    case 'missing':
      return 'Koordinat Kosong';
    case 'partial':
      return 'Parsial';
    case 'invalid':
      return 'Tidak Valid';
    case 'zero':
      return 'Titik Nol (0, 0)';
    case 'duplicate':
      return 'Koordinat Duplikat';
    default:
      return status;
  }
}

export function coordinateIssueLabel(issue: string): string {
  switch (issue) {
    case 'missing_coordinate':
      return 'Koordinat belum diisi';
    case 'partial_coordinate':
      return 'Hanya satu sumbu terisi';
    case 'invalid_coordinate':
      return 'Di luar rentang WGS84';
    case 'zero_coordinate':
      return 'Titik (0, 0)';
    case 'estimated_coordinate':
      return 'Masih estimasi / belum survei';
    case 'duplicate_coordinate':
      return 'Titik koordinat duplikat';
    default:
      return issue;
  }
}

export function formatCoordinatePair(lat: number | null, lng: number | null): string {
  if (lat === null && lng === null) {
    return 'Belum diisi';
  }
  const latStr = lat === null || !Number.isFinite(lat) ? '—' : lat.toFixed(6);
  const lngStr = lng === null || !Number.isFinite(lng) ? '—' : lng.toFixed(6);
  return `${latStr}, ${lngStr}`;
}
