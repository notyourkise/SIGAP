'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { apiRequest, clearAuthCookies } from '@/lib/api';

export interface User {
  id: string;
  username: string;
  full_name: string;
  email: string;
  role: 'admin' | 'teknisi';
  must_change_password?: boolean;
  created_at?: string;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (user: User) => void;
  logout: (reason?: string) => void;
  updateUser: (user: User) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const router = useRouter();

  // Sesi disimpan dalam cookie httpOnly (tidak dapat dibaca JavaScript).
  // Saat aplikasi dimuat, profil pengguna diambil dari endpoint /auth/me.
  useEffect(() => {
    let cancelled = false;

    const bootstrap = async () => {
      try {
        // skipAuthRedirect: true → 401 pada bootstrap adalah kondisi normal
        // (belum login), BUKAN sesi expired. Tanpa flag ini, handleUnauthorized
        // akan terpicu dan membuat phantom cookie yang merusak login berikutnya.
        const response = await apiRequest<User>('/auth/me', { skipAuthRedirect: true });
        if (!cancelled && response.status === 'success' && response.data) {
          setUser(response.data);
          // Paksa ganti password saat login pertama (akun baru / password di-reset
          // Admin) sebelum mengakses menu lain.
          if (response.data.must_change_password && typeof window !== 'undefined') {
            const path = window.location.pathname;
            if (path !== '/change-password' && !path.startsWith('/change-password')) {
              router.replace('/change-password');
            }
          }
        }
      } catch {
        // 401 / jaringan gagal — tidak ada sesi aktif.
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    bootstrap();

    return () => {
      cancelled = true;
    };
  }, []);

  // Sesi berakhir di sisi server (401) — bersihkan state user.
  useEffect(() => {
    const onAuthExpired = () => {
      setUser(null);
    };
    window.addEventListener('sigap:auth-expired', onAuthExpired);
    return () => window.removeEventListener('sigap:auth-expired', onAuthExpired);
  }, []);

  const logout = async (reason?: string) => {
    if (reason) {
      localStorage.setItem('sigap_logout_reason', reason);
    }

    try {
      // Cookie sesi dikirim otomatis; backend me-revoke token & menghapus cookie.
      await apiRequest('/auth/logout', { method: 'POST' });
    } catch {
      // Jika request gagal, tetap lanjutkan proses logout lokal.
    }

    // Hapus cookie CSRF di sisi klien sebagai cadangan (cookie sesi httpOnly
    // hanya bisa dihapus oleh server via Set-Cookie).
    clearAuthCookies();

    setUser(null);
    router.push('/login');
  };

  const login = (newUser: User) => {
    setUser(newUser);
    localStorage.removeItem('sigap_logout_reason');
    // Akun baru / password di-reset → wajib ganti password dulu.
    if (newUser.must_change_password) {
      router.push('/change-password');
    } else {
      router.push('/dashboard');
    }
  };

  const updateUser = (updatedUser: User) => {
    setUser((current) => (current ? { ...current, ...updatedUser } : updatedUser));
  };

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, loading, login, logout, updateUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth harus digunakan di dalam AuthProvider');
  }
  return context;
}
