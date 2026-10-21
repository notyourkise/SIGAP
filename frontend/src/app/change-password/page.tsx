'use client';

export const dynamic = 'force-dynamic';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Eye, EyeOff, LoaderCircle } from 'lucide-react';
import { useAuth, type User as AuthUser } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';

interface ChangePasswordResponse {
  csrf_token: string;
  user: AuthUser;
}

export default function ChangePasswordPage() {
  const router = useRouter();
  const { user, loading: authLoading, logout, updateUser } = useAuth();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const forced = user?.must_change_password === true;
  const passwordIsStrong = newPassword.length >= 8
    && /[A-Z]/.test(newPassword)
    && /[a-z]/.test(newPassword)
    && /\d/.test(newPassword)
    && /[^A-Za-z0-9]/.test(newPassword);
  const passwordsMatch = confirmPassword.length > 0 && newPassword === confirmPassword;

  useEffect(() => {
    if (!authLoading && !user) router.replace('/login');
  }, [authLoading, router, user]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');

    if (!forced && oldPassword.length === 0) {
      setError('Kata sandi saat ini wajib diisi.');
      return;
    }
    if (!passwordIsStrong) {
      setError('Gunakan minimal 8 karakter dengan huruf besar, huruf kecil, angka, dan simbol.');
      return;
    }
    if (!passwordsMatch) {
      setError('Konfirmasi kata sandi belum sama.');
      return;
    }

    setSubmitting(true);
    try {
      const response = await apiRequest<ChangePasswordResponse>('/auth/change-password', {
        method: 'POST',
        body: JSON.stringify({
          ...(forced ? {} : { old_password: oldPassword }),
          new_password: newPassword,
        }),
      });

      if (response.status !== 'success' || !response.data?.user) {
        throw new Error(response.message || 'Kata sandi belum dapat diubah.');
      }

      updateUser(response.data.user);
      router.replace(forced ? '/dashboard' : '/profile?password_changed=true');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Kata sandi belum dapat diubah. Coba lagi.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleBackToLogin = () => {
    void logout();
  };

  if (authLoading || !user) {
    return (
      <main className="flex min-h-screen items-center justify-center gap-3 bg-background" role="status" aria-live="polite">
        <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
        <span className="text-sm text-muted">Memeriksa akun…</span>
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-5 py-10">
      <section aria-labelledby="change-password-heading" className="w-full max-w-md rounded-2xl border border-border bg-card px-6 py-8 sm:px-8">
        <header className="text-center">
          <p className="text-sm font-bold text-primary">SIGAP</p>
          <h1 id="change-password-heading" className="mt-2 text-2xl font-bold tracking-[-0.025em] text-foreground">
            Atur ulang kata sandi
          </h1>
          <p className="mt-2 text-sm text-muted">
            {forced
              ? 'Buat kata sandi baru untuk melanjutkan.'
              : 'Masukkan kata sandi saat ini, lalu buat kata sandi baru.'}
          </p>
        </header>

        {forced && (
          <p role="status" className="mt-4 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning">
            Anda wajib mengganti kata sandi sebelum melanjutkan.
          </p>
        )}

        {error && (
          <div role="alert" aria-live="assertive" className="mt-6 rounded-xl border border-error/30 bg-error/10 px-4 py-3 text-sm text-error">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="mt-7 space-y-5">
          {!forced && (
            <PasswordField
              id="old_password"
              label="Kata sandi saat ini"
              value={oldPassword}
              onChange={setOldPassword}
              visible={showPassword}
              onToggleVisibility={() => setShowPassword((visible) => !visible)}
              disabled={submitting}
              autoFocus
              autoComplete="current-password"
            />
          )}

          <PasswordField
            id="new_password"
            label="Kata sandi baru"
            value={newPassword}
            onChange={setNewPassword}
            visible={showPassword}
            onToggleVisibility={() => setShowPassword((visible) => !visible)}
            disabled={submitting}
            autoFocus={forced}
            hint="Minimal 8 karakter dengan huruf besar, huruf kecil, angka, dan simbol."
          />

          <PasswordField
            id="confirm_password"
            label="Konfirmasi kata sandi baru"
            value={confirmPassword}
            onChange={setConfirmPassword}
            visible={showPassword}
            onToggleVisibility={() => setShowPassword((visible) => !visible)}
            disabled={submitting}
            error={confirmPassword.length > 0 && !passwordsMatch ? 'Konfirmasi belum sama.' : undefined}
          />

          <button
            type="submit"
            disabled={submitting || !passwordIsStrong || !passwordsMatch}
            className="inline-flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-primary px-5 py-3 font-bold text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-not-allowed disabled:opacity-55"
          >
            {submitting ? <><LoaderCircle className="h-4 w-4 animate-spin" /> Menyimpan…</> : 'Simpan kata sandi'}
          </button>
        </form>

        <button
          type="button"
          onClick={handleBackToLogin}
          disabled={submitting}
          className="mx-auto mt-5 flex items-center gap-2 text-sm font-semibold text-muted transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary disabled:cursor-wait disabled:opacity-50"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Kembali ke halaman masuk
        </button>
      </section>
    </main>
  );
}

function PasswordField({
  id,
  label,
  value,
  onChange,
  visible,
  onToggleVisibility,
  disabled,
  autoFocus,
  autoComplete,
  hint,
  error,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  visible: boolean;
  onToggleVisibility: () => void;
  disabled: boolean;
  autoFocus?: boolean;
  autoComplete?: string;
  hint?: string;
  error?: string;
}) {
  const descriptionId = hint || error ? `${id}_description` : undefined;
  return (
    <div>
      <label htmlFor={id} className="mb-2 block text-sm font-semibold text-foreground">{label}</label>
      <div className="relative">
        <input
          id={id}
          type={visible ? 'text' : 'password'}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          autoComplete={autoComplete ?? 'new-password'}
          disabled={disabled}
          autoFocus={autoFocus}
          aria-describedby={descriptionId}
          aria-invalid={Boolean(error)}
          className="w-full rounded-xl border border-input-border bg-input px-4 py-3 pr-12 text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-wait disabled:opacity-60"
          required
        />
        <button
          type="button"
          onClick={onToggleVisibility}
          disabled={disabled}
          aria-label={visible ? `Sembunyikan ${label.toLowerCase()}` : `Tampilkan ${label.toLowerCase()}`}
          className="absolute inset-y-0 right-0 flex items-center pr-4 text-muted transition-colors hover:text-foreground disabled:opacity-50"
        >
          {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </button>
      </div>
      {(error || hint) && (
        <p id={descriptionId} className={`mt-1.5 text-xs ${error ? 'text-error' : 'text-muted'}`}>
          {error || hint}
        </p>
      )}
    </div>
  );
}
