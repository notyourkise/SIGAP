import * as SecureStore from 'expo-secure-store';

// Key penyimpanan token sesi JWT di expo-secure-store.
const TOKEN_KEY = 'sigap_session_token';
const USER_KEY = 'sigap_user';

/**
 * Menyimpan token sesi JWT secara aman. Pada login, backend menetapkan token
 * pada cookie httpOnly `sigap_session`. Untuk klien mobile (bukan browser),
 * kita menangkap nilai token dari header Set-Cookie lalu menyimpannya di sini
 * dan mengirimkannya ulang sebagai `Authorization: Bearer <token>`.
 */
export async function storeSessionToken(token: string): Promise<void> {
  await SecureStore.setItemAsync(TOKEN_KEY, token);
}

export async function getSessionToken(): Promise<string | null> {
  return SecureStore.getItemAsync(TOKEN_KEY);
}

export async function clearSessionToken(): Promise<void> {
  await SecureStore.deleteItemAsync(TOKEN_KEY);
  await SecureStore.deleteItemAsync(USER_KEY);
}

/** Menyimpan profil user yang sedang login (untuk auto-login). */
export async function storeUser(user: unknown): Promise<void> {
  if (!user || typeof user !== 'object') {
    return;
  }
  await SecureStore.setItemAsync(USER_KEY, JSON.stringify(user));
}

export async function getUser<T = unknown>(): Promise<T | null> {
  const raw = await SecureStore.getItemAsync(USER_KEY);
  if (!raw || raw === 'undefined' || raw === 'null') return null;
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') return null;
    return parsed as T;
  } catch {
    return null;
  }
}

/** Menyimpan preferensi tema (tidak kritis, boleh di AsyncStorage/SecureStore). */
export async function storeTheme(theme: string): Promise<void> {
  await SecureStore.setItemAsync('sigap_theme', theme);
}

export async function getTheme(): Promise<string | null> {
  return SecureStore.getItemAsync('sigap_theme');
}
