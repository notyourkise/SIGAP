import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { apiClient, apiGet, extractError } from '../api/client';
import type { LoginResult, User } from '../api/types';
import { clearSessionToken, getSessionToken, getUser, storeUser } from './secureStore';

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  isAuthenticated: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
  updateUser: (user: User) => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

/**
 * Mengekstrak objek User dari respons /auth/me secara tahan banting (resilien).
 * Mendukung baik bentuk nested { user: User } maupun bentuk direct User.
 */
export function extractUser(data: unknown): User | null {
  if (!data || typeof data !== 'object') return null;
  const obj = data as Record<string, unknown>;
  if (obj.user && typeof obj.user === 'object') {
    const u = obj.user as Record<string, unknown>;
    if (typeof u.id === 'string' && typeof u.username === 'string') {
      return obj.user as User;
    }
  }
  if (typeof obj.id === 'string' && typeof obj.username === 'string') {
    return obj as unknown as User;
  }
  return null;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // Login: kirim username & password, simpan token + profil user.
  const login = useCallback(async (username: string, password: string) => {
    const res = await apiPostLogin(username, password);
    if (res.status === 'success' && res.data) {
      await storeUser(res.data.user);
      setUser(res.data.user);
      return;
    }
    throw new Error(res.message || 'Login gagal');
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiClient.post('/auth/logout');
    } catch {
      // Abaikan error logout; tetap bersihkan sesi lokal.
    }
    await clearSessionToken();
    setUser(null);
  }, []);

  // Ambil profil terbaru dari /auth/me (auto-login / refresh sesi).
  const refresh = useCallback(async () => {
    try {
      const res = await apiGet<unknown>('/auth/me');
      if (res.status === 'success' && res.data) {
        const u = extractUser(res.data);
        if (u) {
          setUser(u);
          await storeUser(u);
        }
      }
    } catch {
      // Token tidak valid / sesi kedaluwarsa — biarkan user null.
    }
  }, []);

  const updateUser = useCallback(async (next: User) => {
    if (next && typeof next === 'object') {
      setUser(next);
      await storeUser(next);
    }
  }, []);

  // Auto-login: muat profil dari secure-store, lalu validasi via /auth/me.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const token = await getSessionToken();
      if (!token) {
        if (!cancelled) setLoading(false);
        return;
      }
      const cached = await getUser<User>();
      if (cached && !cancelled) {
        setUser(cached);
      }
      try {
        const res = await apiGet<unknown>('/auth/me');
        if (!cancelled && res.status === 'success' && res.data) {
          const u = extractUser(res.data);
          if (u) {
            setUser(u);
            await storeUser(u);
          }
        }
      } catch {
        // Token kedaluwarsa — bersihkan sesi.
        if (!cancelled) {
          await clearSessionToken();
          setUser(null);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({ user, loading, isAuthenticated: !!user, login, logout, refresh, updateUser }),
    [user, loading, login, logout, refresh, updateUser]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth harus dipakai di dalam AuthProvider');
  }
  return ctx;
}

// Login helper (menangkap Set-Cookie session + menyimpan token).
async function apiPostLogin(username: string, password: string): Promise<{ status: string; message?: string; data?: LoginResult }> {
  try {
    const res = await apiClient.post<{ status: string; message?: string; data?: LoginResult }>(
      '/auth/login',
      { username, password }
    );
    // Token sesi disimpan interceptor dari header Set-Cookie.
    return res.data;
  } catch (error) {
    throw new Error(extractError(error));
  }
}
