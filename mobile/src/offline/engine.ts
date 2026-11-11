import type { CreateFieldMaintenancePayload, FieldCheckItem } from '../api/types';

export const DRAFT_KEY = 'sigap_field_drafts';
export const MAX_DRAFT_RETRY_COUNT = 5;

export type DraftStatus = 'pending' | 'syncing' | 'failed' | 'failed_permanent';

export interface FieldDraft {
  id: string;
  clientRequestId: string;
  userId: string;
  createdAt: string;
  cctvId: string;
  cctvName?: string;
  cctvCode?: string;
  notes: string;
  location?: { lat: string; lng: string } | null;
  checks: FieldCheckItem[];
  photos: string[];
  uploadedImageUrls: string[];
  retryCount: number;
  lastError?: string;
  status: DraftStatus;
}

export type CreateFieldDraftInput = Omit<
  FieldDraft,
  'id' | 'createdAt' | 'clientRequestId' | 'uploadedImageUrls' | 'retryCount' | 'status'
>;

export interface DraftStorageAdapter {
  getItem(key: string): Promise<string | null>;
  setItem(key: string, value: string): Promise<void>;
  removeItem(key: string): Promise<void>;
}

export interface DraftApiAdapter {
  uploadPhoto(base64: string, draft: FieldDraft, index: number): Promise<string>;
  submitFieldMaintenance(
    payload: CreateFieldMaintenancePayload
  ): Promise<{ status: string; message?: string }>;
}

export type SyncErrorKind = 'transient' | 'auth' | 'permanent';

export interface ClassifiedSyncError {
  kind: SyncErrorKind;
  status?: number;
  message: string;
}

export interface SyncAllResult {
  success: number;
  failed: number;
  permanentFailed: number;
  authRequired: boolean;
  total: number;
}

function defaultExtractMessage(error: unknown): string {
  if (error && typeof error === 'object') {
    const maybeAxios = error as {
      response?: { data?: { message?: unknown } };
      message?: unknown;
      code?: unknown;
    };
    const respMsg = maybeAxios.response?.data?.message;
    if (typeof respMsg === 'string' && respMsg.trim()) {
      return respMsg;
    }
    if (maybeAxios.code === 'ECONNABORTED') {
      return 'Waktu permintaan habis. Periksa koneksi Anda.';
    }
    if (typeof maybeAxios.message === 'string' && maybeAxios.message.trim()) {
      return maybeAxios.message;
    }
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return 'Terjadi kesalahan jaringan.';
}

/**
 * Mengklasifikasikan kegagalan sinkronisasi agar error validasi / izin (4xx)
 * atau sesi kedaluwarsa (401) tidak disalahartikan sebagai masalah jaringan offline.
 */
export function classifySyncError(
  error: unknown,
  extractMessage: (err: unknown) => string = defaultExtractMessage
): ClassifiedSyncError {
  const status =
    error && typeof error === 'object'
      ? ((error as { response?: { status?: number }; status?: number }).response?.status ??
          (error as { status?: number }).status)
      : undefined;

  const message = extractMessage(error);

  if (status === 401) {
    return {
      kind: 'auth',
      status,
      message: message || 'Sesi Anda berakhir. Masuk kembali untuk menyinkronkan draf.',
    };
  }

  if (
    status === 400 ||
    status === 403 ||
    status === 404 ||
    status === 409 ||
    status === 422
  ) {
    return {
      kind: 'permanent',
      status,
      message,
    };
  }

  return {
    kind: 'transient',
    status,
    message,
  };
}

function generateLocalId(prefix: string): string {
  const randomUuid =
    typeof globalThis.crypto?.randomUUID === 'function'
      ? globalThis.crypto.randomUUID()
      : `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
  return `${prefix}-${randomUuid}`;
}

export function createDraftStore(
  storage: DraftStorageAdapter,
  api: DraftApiAdapter,
  extractMessage: (err: unknown) => string = defaultExtractMessage
) {
  let mutationChain: Promise<unknown> = Promise.resolve();
  const activeDraftLocks = new Set<string>();
  const activeSyncByUser = new Map<string, Promise<SyncAllResult>>();

  function withStorageLock<T>(fn: () => Promise<T>): Promise<T> {
    const run = mutationChain.then(fn, fn);
    mutationChain = run.then(
      () => undefined,
      () => undefined
    );
    return run;
  }

  async function readAllRaw(): Promise<FieldDraft[]> {
    const raw = await storage.getItem(DRAFT_KEY);
    if (!raw) return [];
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? (parsed as FieldDraft[]) : [];
    } catch {
      return [];
    }
  }

  async function getDrafts(): Promise<FieldDraft[]> {
    return withStorageLock(readAllRaw);
  }

  async function getDraftsByUser(userId: string): Promise<FieldDraft[]> {
    const trimmed = userId.trim();
    if (!trimmed) {
      return [];
    }
    const drafts = await getDrafts();
    return drafts.filter((d) => d.userId === trimmed);
  }

  async function saveDraft(input: CreateFieldDraftInput): Promise<FieldDraft> {
    const ownerId = input.userId?.trim();
    if (!ownerId) {
      throw new Error('Identitas pengguna diperlukan untuk menyimpan draf lokal.');
    }
    const draft: FieldDraft = {
      ...input,
      userId: ownerId,
      id: generateLocalId('draft'),
      clientRequestId: generateLocalId('req'),
      createdAt: new Date().toISOString(),
      uploadedImageUrls: [],
      retryCount: 0,
      status: 'pending',
    };

    await withStorageLock(async () => {
      const drafts = await readAllRaw();
      await storage.setItem(DRAFT_KEY, JSON.stringify([...drafts, draft]));
    });

    return draft;
  }

  async function updateDraft(draft: FieldDraft): Promise<void> {
    await withStorageLock(async () => {
      const drafts = await readAllRaw();
      const next = drafts.map((d) => (d.id === draft.id ? draft : d));
      await storage.setItem(DRAFT_KEY, JSON.stringify(next));
    });
  }

  async function removeDraft(id: string, userId?: string): Promise<FieldDraft[]> {
    return withStorageLock(async () => {
      const drafts = await readAllRaw();
      const next = drafts.filter((d) => {
        if (d.id !== id) return true;
        if (userId && d.userId !== userId) return true;
        return false;
      });
      await storage.setItem(DRAFT_KEY, JSON.stringify(next));
      return next;
    });
  }

  async function clearDrafts(): Promise<void> {
    await withStorageLock(async () => {
      await storage.removeItem(DRAFT_KEY);
    });
  }

  async function syncSingleDraft(draft: FieldDraft): Promise<boolean> {
    if (!draft.userId?.trim()) {
      throw new Error('Draf tidak memiliki identitas teknisi yang valid.');
    }
    if (activeDraftLocks.has(draft.id)) {
      return false;
    }

    activeDraftLocks.add(draft.id);
    const working: FieldDraft = {
      ...draft,
      uploadedImageUrls: [...(draft.uploadedImageUrls || [])],
      status: 'syncing',
    };

    try {
      await updateDraft(working);

      for (let i = working.uploadedImageUrls.length; i < working.photos.length; i++) {
        const url = await api.uploadPhoto(working.photos[i], working, i);
        if (!url) {
          throw new Error('Gagal mengunggah foto bukti');
        }
        working.uploadedImageUrls.push(url);
        await updateDraft(working);
      }

      const payload: CreateFieldMaintenancePayload = {
        cctv_id: working.cctvId,
        notes: working.notes,
        check_latitude: working.location?.lat ?? '',
        check_longitude: working.location?.lng ?? '',
        client_request_id: working.clientRequestId,
        checks: working.checks,
        images: working.uploadedImageUrls.map((url) => ({ image_url: url })),
      };

      const res = await api.submitFieldMaintenance(payload);
      if (res.status === 'success') {
        await removeDraft(working.id, working.userId);
        return true;
      }

      throw Object.assign(new Error(res.message || 'Gagal menyimpan pengecekan lapangan'), {
        status: 400,
      });
    } catch (err: unknown) {
      const classified = classifySyncError(err, extractMessage);
      working.lastError = classified.message;

      if (classified.kind === 'auth') {
        working.status = 'pending';
      } else if (classified.kind === 'permanent') {
        working.status = 'failed_permanent';
      } else {
        working.retryCount = (working.retryCount || 0) + 1;
        working.status = 'failed';
      }

      await updateDraft(working);
      throw err;
    } finally {
      activeDraftLocks.delete(draft.id);
    }
  }

  async function syncAllDrafts(userId: string): Promise<SyncAllResult> {
    const trimmed = userId.trim();
    if (!trimmed) {
      return {
        success: 0,
        failed: 0,
        permanentFailed: 0,
        authRequired: false,
        total: 0,
      };
    }

    const inFlight = activeSyncByUser.get(trimmed);
    if (inFlight) {
      return inFlight;
    }

    const run = (async (): Promise<SyncAllResult> => {
      const targetDrafts = await getDraftsByUser(trimmed);

      let successCount = 0;
      let failedCount = 0;
      let permanentFailedCount = 0;
      let authRequired = false;

      for (const draft of targetDrafts) {
        if (draft.status === 'failed_permanent') {
          permanentFailedCount++;
          continue;
        }
        if ((draft.retryCount || 0) >= MAX_DRAFT_RETRY_COUNT) {
          failedCount++;
          continue;
        }

        try {
          const ok = await syncSingleDraft(draft);
          if (ok) {
            successCount++;
          } else {
            failedCount++;
          }
        } catch (err: unknown) {
          const classified = classifySyncError(err, extractMessage);
          if (classified.kind === 'auth') {
            authRequired = true;
            failedCount++;
            break;
          } else if (classified.kind === 'permanent') {
            permanentFailedCount++;
          } else {
            failedCount++;
          }
        }
      }

      return {
        success: successCount,
        failed: failedCount,
        permanentFailed: permanentFailedCount,
        authRequired,
        total: targetDrafts.length,
      };
    })();

    activeSyncByUser.set(trimmed, run);
    try {
      return await run;
    } finally {
      activeSyncByUser.delete(trimmed);
    }
  }

  return {
    getDrafts,
    getDraftsByUser,
    saveDraft,
    updateDraft,
    removeDraft,
    clearDrafts,
    syncSingleDraft,
    syncAllDrafts,
  };
}
