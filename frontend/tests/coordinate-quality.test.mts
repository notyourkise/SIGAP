import assert from 'node:assert/strict';
import test from 'node:test';

import {
  coordinateIssueLabel,
  coordinateStatusLabel,
  filterCoordinateQualityItems,
  formatCoordinatePair,
  type CoordinateQualityItem,
} from '../src/lib/coordinate-quality.ts';

const sampleItems: CoordinateQualityItem[] = [
  {
    id: '1',
    kode_cctv: 'THD-01',
    nama: 'Rest Area 1',
    lokasi_area: 'Rest Area',
    status: 'aktif',
    latitude: -0.9663481,
    longitude: 116.7207729,
    keterangan: 'Sumber data',
    coordinate_status: 'verified',
    issues: [],
  },
  {
    id: '2',
    kode_cctv: 'THD-18',
    nama: 'Jalan Tol 1C',
    lokasi_area: 'KIPP 1C',
    status: 'nonaktif',
    latitude: -0.972125,
    longitude: 116.7103708,
    keterangan: 'Koordinat: Estimasi',
    coordinate_status: 'estimated',
    issues: ['estimated_coordinate'],
  },
  {
    id: '3',
    kode_cctv: 'THD-05',
    nama: 'Bundaran Sumbu Barat',
    lokasi_area: 'Sumbu Barat',
    status: 'aktif',
    latitude: -0.977067574,
    longitude: 116.7091789,
    keterangan: 'Sumber data',
    coordinate_status: 'duplicate',
    issues: ['duplicate_coordinate'],
  },
  {
    id: '4',
    kode_cctv: 'CAM-MISS',
    nama: 'Kamera Kosong',
    lokasi_area: 'Zona Baru',
    status: 'nonaktif',
    latitude: null,
    longitude: null,
    keterangan: '',
    coordinate_status: 'missing',
    issues: ['missing_coordinate'],
  },
  {
    id: '5',
    kode_cctv: 'CAM-EST-DUP',
    nama: 'Kamera Estimasi & Duplikat',
    lokasi_area: 'KIPP 1B',
    status: 'nonaktif',
    latitude: -0.977067574,
    longitude: 116.7091789,
    keterangan: 'Perkiraan titik awal',
    coordinate_status: 'estimated',
    issues: ['duplicate_coordinate', 'estimated_coordinate'],
  },
];

test('filterCoordinateQualityItems memisahkan kelompok masalah secara akurat', () => {
  assert.deepEqual(
    filterCoordinateQualityItems(sampleItems, 'needs_attention').map((i) => i.kode_cctv),
    ['THD-18', 'THD-05', 'CAM-MISS', 'CAM-EST-DUP']
  );

  assert.deepEqual(
    filterCoordinateQualityItems(sampleItems, 'estimated').map((i) => i.kode_cctv),
    ['THD-18', 'CAM-EST-DUP']
  );

  assert.deepEqual(
    filterCoordinateQualityItems(sampleItems, 'missing_invalid').map((i) => i.kode_cctv),
    ['CAM-MISS']
  );

  assert.deepEqual(
    filterCoordinateQualityItems(sampleItems, 'duplicate').map((i) => i.kode_cctv),
    ['THD-05', 'CAM-EST-DUP']
  );

  assert.equal(filterCoordinateQualityItems(sampleItems, 'all').length, 5);
});

test('label & pemformat koordinat konsisten untuk UI audit', () => {
  assert.equal(coordinateStatusLabel('verified'), 'Terverifikasi');
  assert.equal(coordinateStatusLabel('estimated'), 'Estimasi');
  assert.equal(coordinateIssueLabel('duplicate_coordinate'), 'Titik koordinat duplikat');
  assert.equal(formatCoordinatePair(null, null), 'Belum diisi');
  assert.equal(formatCoordinatePair(-0.9663481, 116.7207729), '-0.966348, 116.720773');
});
