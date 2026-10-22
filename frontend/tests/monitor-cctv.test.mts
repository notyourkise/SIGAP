import assert from 'node:assert/strict';
import test from 'node:test';

import {
  getSelectableCctvs,
  sortCctvsByCode,
  uniqueSelectionForCells,
} from '../src/lib/monitor-cctv.ts';

const cctvs = [
  { id: '15', kode_cctv: 'THD-15', nama: 'Belakang Istana' },
  { id: '2', kode_cctv: 'THD-02', nama: 'Depan Masjid' },
  { id: '10', kode_cctv: 'THD-10', nama: 'Sentra Massa' },
  { id: '3', kode_cctv: 'THD-03', nama: 'Parkiran TKB' },
];

test('kode CCTV diurutkan natural dari THD awal ke akhir', () => {
  assert.deepEqual(
    sortCctvsByCode(cctvs).map((cctv) => cctv.kode_cctv),
    ['THD-02', 'THD-03', 'THD-10', 'THD-15']
  );
});

test('kamera yang dipilih di sel lain tidak muncul lagi', () => {
  const options = getSelectableCctvs(cctvs, { 0: '15', 1: '2' }, 2, [0, 1, 2, 3]);

  assert.deepEqual(
    options.map((cctv) => cctv.id),
    ['10', '3']
  );
});

test('pilihan sel sendiri tetap tersedia sebagai nilai aktif', () => {
  const options = getSelectableCctvs(cctvs, { 0: '15', 1: '2' }, 0, [0, 1, 2, 3]);

  assert.ok(options.some((cctv) => cctv.id === '15'));
  assert.ok(!options.some((cctv) => cctv.id === '2'));
});

test('pilihan pada sel yang tidak terlihat tidak mengunci dropdown', () => {
  const options = getSelectableCctvs(cctvs, { 8: '15' }, 0, [0, 1, 2, 3]);

  assert.equal(options.length, cctvs.length);
});

test('selection lama dibersihkan dari duplikat dan sel tersembunyi', () => {
  const normalized = uniqueSelectionForCells({ 0: '15', 1: '15', 2: '2', 8: '10' }, [0, 1, 2, 3]);

  assert.deepEqual(normalized, { 0: '15', 2: '2' });
});
