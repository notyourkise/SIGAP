import AsyncStorage from '@react-native-async-storage/async-storage';
import { apiClient, extractError } from '../api/client';
import type { CreateFieldMaintenancePayload } from '../api/types';
import {
  classifySyncError,
  createDraftStore,
  DRAFT_KEY,
  MAX_DRAFT_RETRY_COUNT,
  type CreateFieldDraftInput,
  type DraftStatus,
  type FieldDraft,
  type SyncAllResult,
} from './engine';

export {
  classifySyncError,
  createDraftStore,
  DRAFT_KEY,
  MAX_DRAFT_RETRY_COUNT,
  type CreateFieldDraftInput,
  type DraftStatus,
  type FieldDraft,
  type SyncAllResult,
};

/**
 * Membangun bagian FormData untuk unggahan foto (React Native memakai objek
 * { uri, name, type } — base64 di-encode sebagai data URI).
 */
export function buildUploadPart(base64: string): { uri: string; name: string; type: string } {
  return {
    uri: `data:image/jpeg;base64,${base64}`,
    name: `proof-${Date.now()}-${Math.random().toString(36).slice(2, 7)}.jpg`,
    type: 'image/jpeg',
  };
}

const defaultStore = createDraftStore(
  AsyncStorage,
  {
    async uploadPhoto(base64: string) {
      const form = new FormData();
      form.append('file', buildUploadPart(base64) as unknown as Blob);
      const res = await apiClient.post<{
        status: string;
        data?: { url: string };
        message?: string;
      }>('/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      const url = res.data?.data?.url;
      if (!url) {
        throw new Error(res.data?.message || 'Gagal mengunggah foto bukti');
      }
      return url;
    },
    async submitFieldMaintenance(payload: CreateFieldMaintenancePayload) {
      const res = await apiClient.post<{ status: string; message?: string }>(
        '/field-maintenance',
        payload
      );
      return res.data;
    },
  },
  extractError
);

export const getDrafts = defaultStore.getDrafts;
export const getDraftsByUser = defaultStore.getDraftsByUser;
export const saveDraft = defaultStore.saveDraft;
export const updateDraft = defaultStore.updateDraft;
export const removeDraft = defaultStore.removeDraft;
export const clearDrafts = defaultStore.clearDrafts;
export const syncSingleDraft = defaultStore.syncSingleDraft;
export const syncAllDrafts = defaultStore.syncAllDrafts;
