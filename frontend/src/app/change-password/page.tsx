'use client';

// Force dynamic rendering for authenticated-only pages to avoid static export errors
export const dynamic = 'force-dynamic';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Eye, EyeOff, KeyRound, LoaderCircle, Lock, Moon, ShieldAlert, Sun } from 'lucide-react';
import { useAuth, type User as AuthUser } from '@/context/AuthContext';
import { useTheme } from '@/context/ThemeContext';
import { apiRequest } from '@/lib/api';

export default function ChangePasswordPage() {
  const { user, login, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const router = useRouter();
  const searchParams = useSearchParams();
  
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // Early guard: Redirect authenticated non-forced users to dashboard immediately
  // This prevents flash of change-password page when user is already logged in normally
  useEffect(() => {
    if (user && !user.must_change_password) {
      router.push('/dashboard');
    }
  }, [user, router]);

  const forced = user?.must_change_password === true;

  // Handle successful password change navigation
  const handleSuccess = (updatedUser: AuthUser) => {
    login(updatedUser);
    
    // For non-forced mode (direct change), return to profile/context
    // For forced mode, just stay or go to login gracefully
    if (!forced) {
      // Try to go back, fallback to profile
      router.replace('/profile?password_changed=true');
    } else {
      // Forced mode complete - redirect to login to restart session cleanly
      router.push('/login?force_logout=true');
    }
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');

    if (!newPassword || !confirmPassword) {
      setError('Seluruh kolom wajib diisi.');
      return;
    }
    if (newPassword.length < 8) {
      setError('Kata sandi baru minimal 8 karakter.');
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('Konfirmasi kata sandi tidak cocok.');
      return;
    }

    setLoading(true);
    try {
      const response = await apiRequest<{ user: AuthUser }>('/auth/change-password', {
        method: 'POST',
        body: JSON.stringify({ new_password: newPassword }),
      });
      if (response.status === 'success' && response.data) {
        handleSuccess(response.data.user);
      } else {
        setError(response.message || 'Kata sandi belum dapat diubah. Periksa kembali isian Anda.');
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Layanan SIGAP belum dapat dihubungi. Periksa koneksi lalu coba lagi.');
    } finally {
      setLoading(false);
    }
  };

  // Exit to login for forced mode users (if they want to abort/reset)
  const exitToLogin = () => {
    // Only show exit button if we came from somewhere specific
    const fromParam = searchParams.get('from');
    if (fromParam) {
      router.push(fromParam);
    } else {
      // Default: log out and go to login
      router.push('/login');
    }
  };

  // Cancel for non-forced mode: go back or to profile
  const cancelToProfile = () => {
    try {
      router.back();
    } catch {
      router.push('/profile');
    }
  };

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 py-12">
      <div className="absolute inset-x-0 top-0 h-1.5 bg-gradient-to-r from-accent via-accent-soft to-primary" />
      <div className="absolute left-0 top-0 h-72 w-72 -translate-x-1/2 -translate-y-1/2 rounded-full bg-accent/10 blur-3xl" />
      <div className="absolute bottom-0 right-0 h-80 w-80 translate-x-1/2 translate-y-1/2 rounded-full bg-primary/10 blur-3xl" />

      <button
        type="button"
        onClick={toggleTheme}
        aria-label={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
        title={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
        className="absolute right-5 top-5 rounded-xl border border-border bg-card p-2.5 shadow-sm transition-colors hover:border-primary"
      >
        {theme === 'light' ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4 text-accent" />}
      </button>

      <div className="z-10 w-full max-w-2xl rounded-3xl border border-border bg-card p-7 shadow-[0_24px_70px_rgba(40,63,36,0.12)] sm:p-8">
        <header className="mb-7 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary">
            <KeyRound className="h-6 w-6" />
          </div>
          <h1 className="text-2xl font-extrabold tracking-tight">Ganti Kata Sandi</h1>
          <p className="mt-2 text-sm text-muted">
            {forced 
              ? 'Demi keamanan, Anda wajib mengganti kata sandi sebelum menggunakan SIGAP.' 
              : 'Perbarui kata sandi akun Anda untuk menjaga keamanan.'}
          </p>
        </header>

        {forced && (
          <div role="alert" className="mb-5 flex items-center gap-3 rounded-xl border border-amber-500/25 bg-amber-500/10 p-3.5 text-xs font-semibold text-amber-600 dark:text-amber-400">
            <ShieldAlert className="h-4 w-4 shrink-0" />
            <span>Kata sandi sementara wajib diganti. Sesi lama di perangkat lain akan dicabut.</span>
          </div>
        )}

        {error && (
          <div role="alert" className="mb-5 flex items-center gap-3 rounded-xl border border-error/20 bg-error/10 p-3.5 text-sm text-error">
            <ShieldAlert className="h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <div>
              <label className="mb-2 block text-sm font-medium" htmlFor="new-password">Kata sandi baru</label>
              <div className="relative">
                <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
                <input
                  id="new-password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="new-password"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  placeholder="Minimal 8 karakter"
                  disabled={loading}
                  className="w-full rounded-xl border border-input-border bg-input py-3 pl-11 pr-12 text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-wait disabled:opacity-70"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((visible) => !visible)}
                  disabled={loading}
                  aria-label={showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
                  className="absolute inset-y-0 right-0 flex items-center pr-4 text-muted transition-colors hover:text-foreground disabled:opacity-50"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              <p className="mt-2 text-xs text-muted">Gunakan kombinasi huruf besar, huruf kecil, angka, dan simbol.</p>
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium" htmlFor="confirm-password">Ulangi kata sandi baru</label>
              <div className="relative">
                <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
                <input
                  id="confirm-password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="new-password"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  placeholder="Ulangi kata sandi baru"
                  disabled={loading}
                  className="w-full rounded-xl border border-input-border bg-input py-3 pl-11 pr-12 text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-wait disabled:opacity-70"
                  required
                />
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3">
            {/* Force mode: Show exit to login button */}
            {forced ? (
              <button
                type="button"
                onClick={exitToLogin}
                disabled={loading}
                className="rounded-xl border border-border px-5 py-3 text-sm font-semibold text-muted transition-colors hover:bg-background hover:text-foreground disabled:opacity-50"
              >
                Kembali ke Login
              </button>
            ) : (
              /* Non-forced mode: Show cancel to profile button */
              <button
                type="button"
                onClick={cancelToProfile}
                disabled={loading}
                className="rounded-xl border border-border px-5 py-3 text-sm font-semibold text-muted transition-colors hover:bg-background hover:text-foreground disabled:opacity-50"
              >
                Batal
              </button>
            )}
            
            <button
              type="submit"
              disabled={loading}
              className="flex items-center justify-center gap-2 rounded-xl bg-primary px-6 py-3 font-semibold text-white shadow-lg shadow-primary/20 transition-all hover:bg-primary-hover active:scale-[0.98] disabled:cursor-wait disabled:scale-100 disabled:opacity-70"
            >
              {loading ? <><LoaderCircle className="h-4 w-4 animate-spin" /> Menyimpan…</> : 'Simpan Kata Sandi'}
            </button>
          </div>
        </form>
      </div>
    </main>
  );
}
