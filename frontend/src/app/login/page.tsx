'use client';

import { useEffect, useState } from 'react';
import Image from 'next/image';
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
      setError('Username atau email dan kata sandi wajib diisi.');
      return;
    }

    setLoading(true);
    try {
      // skipAuthRedirect: true → 401 pada login artinya "kredensial salah",
      // bukan sesi expired. Tanpa flag ini, handleUnauthorized akan terpicu
      // dan membuat phantom cookie yang merusak percobaan login berikutnya.
      const response = await apiRequest<{ user: AuthUser }>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
        skipAuthRedirect: true,
      });
      if (response.status === 'success' && response.data) {
        login(response.data.user);
      } else {
        setError(response.message || 'Akun atau kata sandi tidak sesuai.');
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Layanan SIGAP belum dapat dihubungi. Periksa koneksi lalu coba lagi.');
    } finally {
      setLoading(false);
    }
  };

  if (authLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center gap-3 bg-background" role="status">
        <LoaderCircle className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm text-muted">Memeriksa sesi pengguna…</span>
      </div>
    );
  }

  return (
    <main className="flex min-h-screen flex-col bg-background lg:flex-row">
      {/* Panel kiri — foto malam Kompleks Kemenko I (Garuda Palace) IKN */}
      <div className="relative h-52 w-full shrink-0 sm:h-72 lg:h-screen lg:w-1/2">
        <Image
          src="/assets/foto/ikn-kemenko.webp"
          alt="Kompleks Kemenko I (Garuda Palace) IKN Nusantara di malam hari"
          fill
          priority
          sizes="(min-width: 1024px) 50vw, 100vw"
          className="object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black/45 via-black/5 to-transparent lg:bg-gradient-to-r lg:from-transparent lg:via-black/5 lg:to-black/25" />
      </div>

      {/* Panel kanan — krem lembut / hijau tua saat mode gelap */}
      <div className="relative flex w-full flex-1 items-center justify-center px-6 py-14 sm:px-10 lg:w-1/2 lg:py-16">
        {/* Status icons pojok kanan atas — hanya toggle tema (lonceng notifikasi dihapus, tidak relevan di halaman login) */}
        <div className="absolute right-5 top-5 z-10 flex items-center gap-3">
          <button
            type="button"
            onClick={toggleTheme}
            aria-label={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
            title={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
            className="rounded-xl border border-border bg-card/70 p-2.5 text-foreground shadow-sm transition-colors hover:border-primary"
          >
            {theme === 'light' ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4 text-accent" />}
          </button>
        </div>

        <div className="w-full max-w-sm">
          {/* Logo Pohon Hayat NUSANTARA */}
          <header className="text-center">
            <Image
              src="/nusantara-logo.png"
              alt="Logo Pohon Hayat NUSANTARA"
              width={132}
              height={132}
              priority
              className="mx-auto h-28 w-auto sm:h-32"
            />
            <p className="mt-4 text-[10px] font-bold uppercase tracking-[0.38em] text-[#b98a2f]">NUSANTARA</p>
            <p className="mt-1 text-[10px] font-medium uppercase tracking-[0.22em] text-muted">
              Ibu Kota Negara Indonesia
            </p>

            <div className="mx-auto mt-6 h-px w-24 bg-gradient-to-r from-transparent via-[#d9c08a] to-transparent" />

            <h1 className="mt-6 font-serif text-5xl font-bold tracking-tight text-primary sm:text-[3.25rem]">
              SIGAP
            </h1>
            <p className="mt-2 text-sm text-muted">Sistem internal pengelolaan CCTV kawasan IKN.</p>
          </header>

          {error && (
            <div role="alert" className="mb-5 mt-8 flex items-center gap-3 rounded-xl border border-error/20 bg-error/10 p-3.5 text-sm text-error">
              <ShieldAlert className="h-4 w-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="mt-8 space-y-5">
            <div>
              <label className="mb-2 block text-sm font-medium text-foreground" htmlFor="username">Username atau email</label>
              <div className="relative">
                <User className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
                <input
                  id="username"
                  type="text"
                  autoComplete="username"
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  placeholder="Masukkan username atau email"
                  disabled={loading}
                  className="w-full rounded-xl border border-input-border bg-input py-3 pl-11 pr-4 text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-wait disabled:opacity-70"
                  required
                />
              </div>
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium text-foreground" htmlFor="password">Kata sandi</label>
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
              {loading ? <><LoaderCircle className="h-4 w-4 animate-spin" /> Memeriksa akun…</> : 'Masuk'}
            </button>
          </form>
        </div>
      </div>
    </main>
  );
}
