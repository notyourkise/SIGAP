'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { apiRequest, clearAuthCookies } from '@/lib/api';

export interface User {
  id: string;
  username: string;
  full_name: string;
  email: string;
  role: 'admin' | 'teknisi';
  must_change_password?: boolean;
  created_at?: string;
  profile_photo_url?: string;
  // Preferensi notifikasi email (Task 11.3) — dikelola di halaman profil.
  email_notifications?: boolean;
  // Preferensi email granular per-tipe (Task 13.7, Sprint 32).
  email_daily_digest?: boolean;
  email_overdue?: boolean;
  email_stock_low?: boolean;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (user: User) => void;
  logout: () => void;
  updateUser: (user: User) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

function navigateAfterAuth(path: string) {
  if (typeof window !== 'undefined') {
    // Full navigation membatalkan request halaman lama sehingga 401 stale
    // dari dashboard sebelum logout tidak dapat mencabut sesi baru.
    window.location.replace(path);
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

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

  // Satu sumber kebenaran untuk pengalihan pasca-autentikasi.
  //
  // Pengguna yang wajib mengganti sandi diarahkan ke /change-password.
  // Kunci anti-loop: navigasi HANYA dilakukan bila path saat ini berbeda dari
  // tujuan. Versi lama selalu memanggil window.location.replace('/change-password')
  // setiap kali provider ter-mount ulang — dan karena /change-password adalah
  // halaman penuh, reload tanpa henti pun terjadi (ratusan /auth/me sampai 429).
  useEffect(() => {
    if (loading || !user) return;
    if (user.must_change_password && pathname !== '/change-password') {
      // Navigasi halus (tanpa reload penuh) agar provider tidak ter-mount ulang.
      router.replace('/change-password');
    }
  }, [loading, user, pathname, router]);

  // Sesi berakhir di sisi server (401) — bersihkan state user.
  useEffect(() => {
    const onAuthExpired = () => {
      setUser(null);
    };
    window.addEventListener('sigap:auth-expired', onAuthExpired);
    return () => window.removeEventListener('sigap:auth-expired', onAuthExpired);
  }, []);

  const logout = async () => {
    try {
      // Cookie sesi dikirim otomatis; backend me-revoke token & menghapus cookie.
      await apiRequest('/auth/logout', { method: 'POST', skipAuthRedirect: true });
    } catch {
      // Jika request gagal, tetap lanjutkan proses logout lokal.
    }

    // Hapus cookie CSRF di sisi klien sebagai cadangan (cookie sesi httpOnly
    // hanya bisa dihapus oleh server via Set-Cookie).
    clearAuthCookies();

    setUser(null);
    navigateAfterAuth('/login');
  };

  const login = (newUser: User) => {
    setUser(newUser);
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
