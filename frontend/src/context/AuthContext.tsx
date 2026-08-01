'use client';

import React, { createContext, useContext, useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { apiRequest } from '@/lib/api';
import { Clock } from 'lucide-react';

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
  extendSession: () => void;
  updateUser: (user: User) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Konfigurasi Durasi Idle Timer (15 menit = 900.000 ms, Peringatan pada 14 menit = 840.000 ms)
const IDLE_TIMEOUT_MS = 15 * 60 * 1000;
const WARNING_TIMEOUT_MS = 14 * 60 * 1000;

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // Idle Timer State
  const [showIdleWarning, setShowIdleWarning] = useState(false);
  const [remainingSeconds, setRemainingSeconds] = useState(60);
  const lastActivityRef = useRef<number>(Date.now());
  const lastThrottleRef = useRef<number>(0);

  const router = useRouter();

  // Sesi disimpan dalam cookie httpOnly (tidak dapat dibaca JavaScript).
  // Saat aplikasi dimuat, profil pengguna diambil dari endpoint /auth/me.
  useEffect(() => {
    let cancelled = false;

    const bootstrap = async () => {
      try {
        const response = await apiRequest<User>('/auth/me');
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
      setShowIdleWarning(false);
    };
    window.addEventListener('sigap:auth-expired', onAuthExpired);
    return () => window.removeEventListener('sigap:auth-expired', onAuthExpired);
  }, []);

  // Update aktivitas pengguna saat ada aksi (mousemove, keydown, click, scroll, touchstart)
  useEffect(() => {
    if (!user) return;

    const handleUserActivity = () => {
      const now = Date.now();
      // Throttling: Update timestamp maksimal 1 kali setiap 2 detik
      if (now - lastThrottleRef.current > 2000) {
        lastThrottleRef.current = now;
        lastActivityRef.current = now;
      }
    };

    const activityEvents = ['mousemove', 'keydown', 'click', 'scroll', 'touchstart'];
    activityEvents.forEach((event) => {
      window.addEventListener(event, handleUserActivity, { passive: true });
    });

    return () => {
      activityEvents.forEach((event) => {
        window.removeEventListener(event, handleUserActivity);
      });
    };
  }, [user]);

  const extendSession = () => {
    const now = Date.now();
    lastActivityRef.current = now;
    lastThrottleRef.current = now;
    setShowIdleWarning(false);
  };

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

    setUser(null);
    setShowIdleWarning(false);
    router.push('/login');
  };

  // Interval checker untuk mengecek inaktivitas setiap 1 detik
  useEffect(() => {
    if (!user) {
      setShowIdleWarning(false);
      return;
    }

    const interval = setInterval(() => {
      const now = Date.now();
      const idleTime = now - lastActivityRef.current;

      if (idleTime >= IDLE_TIMEOUT_MS) {
        // Waktu pasif sudah melebihi 15 menit -> Otomatis Logout
        logout('inactivity');
      } else if (idleTime >= WARNING_TIMEOUT_MS) {
        // Waktu pasif sudah mencapai 14 menit -> Tampilkan Peringatan
        setShowIdleWarning(true);
        const secondsLeft = Math.max(0, Math.ceil((IDLE_TIMEOUT_MS - idleTime) / 1000));
        setRemainingSeconds(secondsLeft);
      } else {
        setShowIdleWarning(false);
      }
    }, 1000);

    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user]);

  const login = (newUser: User) => {
    lastActivityRef.current = Date.now();
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
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, loading, login, logout, extendSession, updateUser }}>
      {children}

      {/* Modal Peringatan Inaktivitas (Idle Warning Modal) */}
      {showIdleWarning && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
          <div className="relative w-full max-w-md bg-card border border-border rounded-3xl p-6 shadow-2xl space-y-5 text-foreground">
            <div className="flex items-center gap-3 border-b border-border pb-4">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-amber-500/15 text-amber-600 dark:text-amber-400">
                <Clock className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-lg font-bold">Sesi Hampir Berakhir</h3>
                <p className="text-xs text-muted">Peringatan keamanan inaktivitas</p>
              </div>
            </div>

            <div className="space-y-3 text-sm">
              <p className="text-muted leading-relaxed">
                Anda telah pasif (tidak ada aktivitas) selama hampir 15 menit. Sesi Anda akan otomatis diakhiri demi keamanan dalam:
              </p>
              <div className="flex items-center justify-center gap-2 rounded-2xl bg-amber-500/10 border border-amber-500/20 py-3 text-center">
                <span className="font-mono text-2xl font-black text-amber-600 dark:text-amber-400">
                  00:{remainingSeconds < 10 ? `0${remainingSeconds}` : remainingSeconds}
                </span>
                <span className="text-xs font-semibold text-amber-600 dark:text-amber-400">detik</span>
              </div>
            </div>

            <div className="flex items-center justify-end gap-3 border-t border-border pt-4">
              <button
                type="button"
                onClick={() => logout('inactivity')}
                className="rounded-xl border border-border px-4 py-2.5 text-xs font-semibold text-muted hover:bg-background hover:text-foreground transition-colors"
              >
                Keluar Sekarang
              </button>
              <button
                type="button"
                onClick={extendSession}
                className="rounded-xl bg-primary px-5 py-2.5 text-xs font-semibold text-white shadow-md shadow-primary/20 hover:bg-primary-hover transition-colors"
              >
                Tetap Masuk
              </button>
            </div>
          </div>
        </div>
      )}
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
