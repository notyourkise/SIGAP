import AsyncStorage from '@react-native-async-storage/async-storage';
import type { CreateFieldMaintenancePayload } from '../api/types';

const DRAFT_KEY = 'sigap_field_drafts';

// Draft offline pengecekan lapangan yang disimpan saat sinyal hilang.
export interface FieldDraft {
  id: string;
  createdAt: string;
  payload: CreateFieldMaintenancePayload;
}

export async function getDrafts(): Promise<FieldDraft[]> {
  const raw = await AsyncStorage.getItem(DRAFT_KEY);
  if (!raw) return [];
  try {
    return JSON.parse(raw) as FieldDraft[];
  } catch {
    return [];
  }
}

export async function saveDraft(payload: CreateFieldMaintenancePayload): Promise<FieldDraft[]> {
  const drafts = await getDrafts();
  const draft: FieldDraft = {
    id: `${Date.now()}`,
    createdAt: new Date().toISOString(),
    payload,
  };
  const next = [...drafts, draft];
  await AsyncStorage.setItem(DRAFT_KEY, JSON.stringify(next));
  return next;
}

export async function removeDraft(id: string): Promise<FieldDraft[]> {
  const drafts = await getDrafts();
  const next = drafts.filter((d) => d.id !== id);
  await AsyncStorage.setItem(DRAFT_KEY, JSON.stringify(next));
  return next;
}

export async function clearDrafts(): Promise<void> {
  await AsyncStorage.removeItem(DRAFT_KEY);
}
