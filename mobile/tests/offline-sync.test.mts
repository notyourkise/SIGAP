import assert from 'node:assert/strict';
import test from 'node:test';

import {
  classifySyncError,
  createDraftStore,
  MAX_DRAFT_RETRY_COUNT,
  type DraftApiAdapter,
  type DraftStorageAdapter,
  type FieldDraft,
} from '../src/offline/engine.ts';

function createMemoryStorage(initial?: Record<string, string>): DraftStorageAdapter & {
  dump: () => Record<string, string>;
} {
  const data = new Map<string, string>(Object.entries(initial ?? {}));
  return {
    async getItem(key: string) {
      return data.has(key) ? data.get(key)! : null;
    },
    async setItem(key: string, value: string) {
      data.set(key, value);
    },
    async removeItem(key: string) {
      data.delete(key);
    },
    dump() {
      return Object.fromEntries(data.entries());
    },
  };
}

function makeSampleInput(userId: string, cctvId = '11111111-1111-1111-1111-111111111111') {
  return {
    userId,
    cctvId,
    cctvName: 'Kamera Gerbang KIPP',
    cctvCode: 'CAM-001',
    notes: 'Cek rutin offline',
    location: { lat: '-0.972100', lng: '116.709200' },
    checks: [{ name: 'Pengecekan jaringan', result: 'ok' as const, note: '' }],
    photos: ['base64-photo-1', 'base64-photo-2'],
  };
}

test('draft storage isolates technicians and rejects empty userId', async () => {
  const storage = createMemoryStorage();
  const api: DraftApiAdapter = {
    async uploadPhoto() {
      return '/api/uploads/unused.jpg';
    },
    async submitFieldMaintenance() {
      return { status: 'success' };
    },
  };
  const store = createDraftStore(storage, api);

  await assert.rejects(
    () => store.saveDraft(makeSampleInput('   ')),
    /Identitas pengguna diperlukan/i
  );

  const draftA = await store.saveDraft(makeSampleInput('tech-A', 'cctv-a'));
  const draftB = await store.saveDraft(makeSampleInput('tech-B', 'cctv-b'));

  assert.equal((await store.getDraftsByUser('')).length, 0);
  assert.deepEqual(
    (await store.getDraftsByUser('tech-A')).map((d) => d.id),
    [draftA.id]
  );
  assert.deepEqual(
    (await store.getDraftsByUser('tech-B')).map((d) => d.id),
    [draftB.id]
  );

  // Teknisi B menyinkronkan antreannya — draf milik teknisi A tidak tersentuh
  const resB = await store.syncAllDrafts('tech-B');
  assert.equal(resB.success, 1);
  assert.equal(resB.total, 1);
  assert.equal((await store.getDraftsByUser('tech-B')).length, 0);
  assert.equal((await store.getDraftsByUser('tech-A')).length, 1);

  // Panggilan user kosong aman sebagai no-op
  const emptyRes = await store.syncAllDrafts('');
  assert.deepEqual(emptyRes, {
    success: 0,
    failed: 0,
    permanentFailed: 0,
    authRequired: false,
    total: 0,
  });
});

test('partial photo uploads resume from uploadedImageUrls and preserve client_request_id on retry', async () => {
  const storage = createMemoryStorage();
  const uploadedPayloads: string[] = [];
  const submittedClientRequestIds: Array<string | undefined> = [];
  let failSecondUpload = true;
  let failFirstSubmit = true;

  const api: DraftApiAdapter = {
    async uploadPhoto(base64) {
      uploadedPayloads.push(base64);
      if (base64 === 'base64-photo-2' && failSecondUpload) {
        failSecondUpload = false;
        throw new Error('Network Error');
      }
      return `/api/uploads/${base64}.jpg`;
    },
    async submitFieldMaintenance(payload) {
      submittedClientRequestIds.push(payload.client_request_id);
      if (failFirstSubmit) {
        failFirstSubmit = false;
        throw Object.assign(new Error('Gateway Timeout'), { response: { status: 504 } });
      }
      return { status: 'success' };
    },
  };

  const store = createDraftStore(storage, api);
  const draft = await store.saveDraft(makeSampleInput('tech-A'));

  // Percobaan 1: foto 1 terunggah, foto 2 gagal jaringan -> progress foto 1 tetap tersimpan
  await assert.rejects(() => store.syncSingleDraft(draft));
  let stored = (await store.getDraftsByUser('tech-A'))[0];
  assert.equal(stored.status, 'failed');
  assert.equal(stored.retryCount, 1);
  assert.deepEqual(stored.uploadedImageUrls, ['/api/uploads/base64-photo-1.jpg']);

  // Percobaan 2: foto 1 TIDAK diunggah ulang; foto 2 berhasil; POST /field-maintenance gagal 504
  await assert.rejects(() => store.syncSingleDraft(stored));
  stored = (await store.getDraftsByUser('tech-A'))[0];
  assert.equal(stored.status, 'failed');
  assert.equal(stored.retryCount, 2);
  assert.deepEqual(stored.uploadedImageUrls, [
    '/api/uploads/base64-photo-1.jpg',
    '/api/uploads/base64-photo-2.jpg',
  ]);
  // Jumlah upload sejauh ini: foto1, foto2 (gagal), foto2 (berhasil) — foto 1 tidak diulang!
  assert.deepEqual(uploadedPayloads, ['base64-photo-1', 'base64-photo-2', 'base64-photo-2']);

  // Percobaan 3: semua foto sudah tersimpan, POST berhasil memakai client_request_id yang sama persis
  const res = await store.syncAllDrafts('tech-A');
  assert.equal(res.success, 1);
  assert.equal((await store.getDraftsByUser('tech-A')).length, 0);
  assert.deepEqual(submittedClientRequestIds, [draft.clientRequestId, draft.clientRequestId]);
  assert.deepEqual(uploadedPayloads, ['base64-photo-1', 'base64-photo-2', 'base64-photo-2']);
});

test('error classification: 400 marks failed_permanent, 401 pauses batch without incrementing retryCount, 5xx increments retryCount up to cap', async () => {
  const storage = createMemoryStorage();
  const submittedIds: string[] = [];

  const err400 = Object.assign(new Error('Checklist wajib tidak lengkap'), {
    response: { status: 400, data: { message: 'Checklist wajib tidak lengkap' } },
  });
  const err401 = Object.assign(new Error('Unauthorized'), {
    response: { status: 401, data: { message: 'Sesi habis' } },
  });
  const err500 = Object.assign(new Error('Internal Server Error'), {
    response: { status: 500, data: { message: 'Server error' } },
  });

  assert.equal(classifySyncError(err400).kind, 'permanent');
  assert.equal(classifySyncError(err401).kind, 'auth');
  assert.equal(classifySyncError(err500).kind, 'transient');
  assert.equal(classifySyncError(new Error('Network Error')).kind, 'transient');

  let mode: '400' | '401' | '500' = '400';
  const api: DraftApiAdapter = {
    async uploadPhoto(base64) {
      return `/api/uploads/${base64}.jpg`;
    },
    async submitFieldMaintenance(payload) {
      submittedIds.push(payload.cctv_id);
      if (mode === '400') throw err400;
      if (mode === '401') throw err401;
      throw err500;
    },
  };

  const store = createDraftStore(storage, api);
  const d1 = await store.saveDraft({ ...makeSampleInput('tech-1', 'cctv-400'), photos: ['p1'] });
  const d2 = await store.saveDraft({ ...makeSampleInput('tech-1', 'cctv-401'), photos: ['p2'] });
  const d3 = await store.saveDraft({ ...makeSampleInput('tech-1', 'cctv-after-401'), photos: ['p3'] });

  // Skenario 1: d1 gagal 400 -> failed_permanent, lalu ubah mode ke 401 untuk d2 -> batch berhenti sebelum d3
  await assert.rejects(() => store.syncSingleDraft(d1));
  mode = '401';
  const batchRes = await store.syncAllDrafts('tech-1');

  assert.equal(batchRes.permanentFailed, 1);
  assert.equal(batchRes.authRequired, true);
  assert.equal(batchRes.failed, 1);
  // d3 tidak dikirim karena batch berhenti saat 401
  assert.deepEqual(submittedIds, ['cctv-400', 'cctv-401']);

  const afterBatch = await store.getDraftsByUser('tech-1');
  const mapById = new Map<string, FieldDraft>(afterBatch.map((d) => [d.id, d]));
  assert.equal(mapById.get(d1.id)?.status, 'failed_permanent');
  assert.equal(mapById.get(d2.id)?.status, 'pending');
  assert.equal(mapById.get(d2.id)?.retryCount, 0);
  assert.equal(mapById.get(d3.id)?.status, 'pending');

  // Skenario 2: draft transient 500 melewati MAX_DRAFT_RETRY_COUNT berhenti dicoba ulang
  await store.removeDraft(d2.id, 'tech-1');
  await store.removeDraft(d3.id, 'tech-1');
  const dMax = await store.saveDraft({ ...makeSampleInput('tech-1', 'cctv-500'), photos: ['p4'] });
  await store.updateDraft({ ...dMax, retryCount: MAX_DRAFT_RETRY_COUNT, status: 'failed' });

  mode = '500';
  submittedIds.length = 0;
  const maxRes = await store.syncAllDrafts('tech-1');
  assert.equal(maxRes.permanentFailed, 1);
  assert.equal(maxRes.failed, 1);
  assert.deepEqual(submittedIds, []);
});

test('concurrent syncAllDrafts shares one single-flight run and corrupt storage degrades safely', async () => {
  const corruptStorage = createMemoryStorage({ sigap_field_drafts: '{not-valid-json' });
  let uploadCalls = 0;
  let submitCalls = 0;

  const api: DraftApiAdapter = {
    async uploadPhoto(base64) {
      uploadCalls++;
      await new Promise((r) => setTimeout(r, 15));
      return `/api/uploads/${base64}.jpg`;
    },
    async submitFieldMaintenance() {
      submitCalls++;
      await new Promise((r) => setTimeout(r, 15));
      return { status: 'success' };
    },
  };

  const store = createDraftStore(corruptStorage, api);
  assert.deepEqual(await store.getDraftsByUser('tech-A'), []);

  await store.saveDraft({ ...makeSampleInput('tech-A'), photos: ['only-one'] });

  const [r1, r2] = await Promise.all([
    store.syncAllDrafts('tech-A'),
    store.syncAllDrafts('tech-A'),
  ]);

  assert.equal(uploadCalls, 1);
  assert.equal(submitCalls, 1);
  assert.equal(r1.success, 1);
  assert.equal(r2.success, 1);
  assert.equal((await store.getDraftsByUser('tech-A')).length, 0);
});
