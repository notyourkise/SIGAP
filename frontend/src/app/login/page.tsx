'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Eye, EyeOff, LoaderCircle, Lock, Moon, ShieldAlert, Sun, User } from 'lucide-react';
import { useAuth, type User as AuthUser } from '@/context/AuthContext';
import { useTheme } from '@/context/ThemeContext';
import { apiRequest } from '@/lib/api';

export default function LoginPage() {
  const { login, isAuthenticated, loading: authLoading } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isAuthenticated && !authLoading) router.push('/');
  }, [isAuthenticated, authLoading, router]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');
    if (!username || !password) {
      setError('Username/email dan kata sandi wajib diisi.');
      return;
    }

    setLoading(true);
    try {
      const response = await apiRequest<{ token: string; user: AuthUser }>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      });
      if (response.status === 'success' && response.data) {
        login(response.data.token, response.data.user);
      } else {
        setError(response.message || 'Akun atau kata sandi tidak sesuai.');
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Server belum dapat dihubungi.');
    } finally {
      setLoading(false);
    }
  };

  if (authLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <LoaderCircle className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 py-12">
      <div className="absolute inset-x-0 top-0 h-1.5 bg-gradient-to-r from-accent via-accent-soft to-primary" />
      <div className="absolute left-0 top-0 h-72 w-72 -translate-x-1/2 -translate-y-1/2 rounded-full bg-accent/10 blur-3xl" />
      <div className="absolute bottom-0 right-0 h-80 w-80 translate-x-1/2 translate-y-1/2 rounded-full bg-primary/10 blur-3xl" />

      <button
        type="button"
        onClick={toggleTheme}
        aria-label={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
        title={theme === 'light' ? 'Mode Gelap' : 'Mode Terang'}
        className="absolute right-5 top-5 rounded-xl border border-border bg-card p-2.5 shadow-sm transition-colors hover:border-primary"
      >
        {theme === 'light' ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4 text-accent" />}
      </button>

      <div className="z-10 w-full max-w-sm rounded-3xl border border-border bg-card p-7 shadow-[0_24px_70px_rgba(40,63,36,0.12)] sm:p-8">
        <header className="mb-7 text-center">
          <h1 className="text-3xl font-extrabold tracking-tight text-primary">SIGAP</h1>
          <p className="mt-2 text-sm text-muted">Masuk untuk melanjutkan.</p>
        </header>

        {error && (
          <div role="alert" className="mb-5 flex items-center gap-3 rounded-xl border border-error/20 bg-error/10 p-3.5 text-sm text-error">
            <ShieldAlert className="h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="mb-2 block text-sm font-medium" htmlFor="username">Username atau email</label>
            <div className="relative">
              <User className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
              <input
                id="username"
                type="text"
                autoComplete="username"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                placeholder="Masukkan akun"
                disabled={loading}
                className="w-full rounded-xl border border-input-border bg-input py-3 pl-11 pr-4 text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-wait disabled:opacity-70"
                required
              />
            </div>
          </div>

          <div>
            <label className="mb-2 block text-sm font-medium" htmlFor="password">Kata sandi</label>
            <div className="relative">
              <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="Masukkan kata sandi"
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
          </div>

          <button
            type="submit"
            disabled={loading}
            className="flex w-full items-center justify-center gap-2 rounded-xl bg-primary px-4 py-3.5 font-semibold text-white shadow-lg shadow-primary/20 transition-all hover:bg-primary-hover active:scale-[0.98] disabled:cursor-wait disabled:scale-100 disabled:opacity-70"
          >
            {loading ? <><LoaderCircle className="h-4 w-4 animate-spin" /> Memproses...</> : 'Masuk'}
          </button>
        </form>
      </div>
    </main>
  );
}
