'use client';

import { useState } from 'react';
import Link from 'next/link';
import {
  AtSign,
  BadgeCheck,
  Calendar,
  KeyRound,
  LoaderCircle,
  Mail,
  Save,
  ShieldAlert,
  UserRound,
} from 'lucide-react';
import DashboardLayout from '@/components/DashboardLayout';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';

export default function ProfilePage() {
  const { user, updateUser } = useAuth();
  const [fullName, setFullName] = useState(user?.full_name ?? '');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');
    setSuccess('');

    if (fullName.trim().length < 2) {
      setError('Nama tampilan minimal 2 karakter.');
      return;
    }

    setLoading(true);
    try {
      const response = await apiRequest<{ id: string; username: string; full_name: string; email: string; role: 'admin' | 'teknisi' }>('/auth/profile', {
        method: 'PUT',
        body: JSON.stringify({ full_name: fullName.trim() }),
      });
      if (response.status === 'success' && response.data) {
        updateUser(response.data);
        setSuccess('Profil berhasil diperbarui.');
      } else {
        setError(response.message || 'Profil belum dapat diperbarui. Periksa kembali isian Anda.');
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Layanan SIGAP belum dapat dihubungi. Periksa koneksi lalu coba lagi.');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (value?: string) => {
    if (!value) return '—';
    const date = new Date(value);
    return Number.isNaN(date.getTime())
      ? value
      : date.toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' });
  };

  return (
    <DashboardLayout>
      <div className="space-y-8">
        <header>
          <h1 className="text-3xl font-bold tracking-[-0.025em]">Profil Saya</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted">
            Kelola informasi akun dan keamanan akses SIGAP.
          </p>
        </header>

        {error && (
          <div role="alert" className="flex items-center gap-3 rounded-xl border border-error/20 bg-error/10 p-3.5 text-sm text-error">
            <ShieldAlert className="h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}
        {success && (
          <div role="alert" className="flex items-center gap-3 rounded-xl border border-success/25 bg-success/10 p-3.5 text-sm text-success">
            <BadgeCheck className="h-4 w-4 shrink-0" />
            <span>{success}</span>
          </div>
        )}

        {user && (
          <section aria-labelledby="profile-details-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
            <div className="flex items-center gap-4 border-b border-border px-5 py-5">
              <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-accent/20 text-[#775800] dark:text-accent">
                <UserRound className="h-7 w-7" />
              </div>
              <div className="min-w-0">
                <h2 id="profile-details-heading" className="truncate text-lg font-bold">{user.full_name}</h2>
                <span className={`mt-1 inline-block rounded-md border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider ${
                  user.role === 'admin'
                    ? 'bg-accent/20 text-[#775800] border border-accent/30 dark:text-accent'
                    : 'bg-success/10 text-success border border-success/20'
                }`}>
                  {user.role}
                </span>
              </div>
            </div>

            <dl className="grid gap-px bg-border sm:grid-cols-2">
              <div className="flex items-center gap-3 bg-card px-5 py-4">
                <AtSign className="h-4 w-4 shrink-0 text-muted" />
                <div>
                  <dt className="text-xs font-semibold text-muted">Nama pengguna</dt>
                  <dd className="text-sm font-medium">{user.username}</dd>
                </div>
              </div>
              <div className="flex items-center gap-3 bg-card px-5 py-4">
                <Mail className="h-4 w-4 shrink-0 text-muted" />
                <div>
                  <dt className="text-xs font-semibold text-muted">Email</dt>
                  <dd className="text-sm font-medium">{user.email || '—'}</dd>
                </div>
              </div>
              <div className="flex items-center gap-3 bg-card px-5 py-4">
                <Calendar className="h-4 w-4 shrink-0 text-muted" />
                <div>
                  <dt className="text-xs font-semibold text-muted">Terdaftar sejak</dt>
                  <dd className="text-sm font-medium">{formatDate(user.created_at)}</dd>
                </div>
              </div>
              <div className="flex items-center gap-3 bg-card px-5 py-4">
                <BadgeCheck className="h-4 w-4 shrink-0 text-muted" />
                <div>
                  <dt className="text-xs font-semibold text-muted">Status akun</dt>
                  <dd className="text-sm font-medium">
                    {user.must_change_password ? 'Menunggu ganti kata sandi' : 'Aktif'}
                  </dd>
                </div>
              </div>
            </dl>
          </section>
        )}

        <section aria-labelledby="edit-profile-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="border-b border-border px-5 py-4">
            <h2 id="edit-profile-heading" className="text-base font-bold">Informasi Tampilan</h2>
            <p className="mt-0.5 text-xs text-muted">Nama ini ditampilkan pada sidebar dan aktivitas sistem.</p>
          </div>
          <form onSubmit={handleSubmit} className="space-y-5 p-5">
            <div>
              <label className="mb-2 block text-sm font-medium" htmlFor="full-name">Nama lengkap</label>
              <input
                id="full-name"
                type="text"
                value={fullName}
                onChange={(event) => setFullName(event.target.value)}
                placeholder="Nama lengkap Anda"
                disabled={loading}
                minLength={2}
                maxLength={100}
                className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-wait disabled:opacity-70"
                required
              />
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <button
                type="submit"
                disabled={loading}
                className="flex items-center justify-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-white shadow-md shadow-primary/20 transition-all hover:bg-primary-hover active:scale-[0.98] disabled:cursor-wait disabled:scale-100 disabled:opacity-70"
              >
                {loading ? <><LoaderCircle className="h-4 w-4 animate-spin" /> Menyimpan…</> : <><Save className="h-4 w-4" /> Simpan Perubahan</>}
              </button>
              <Link
                href="/change-password"
                className="flex items-center justify-center gap-2 rounded-xl border border-border px-4 py-2.5 text-sm font-semibold text-muted transition-colors hover:bg-background hover:text-foreground"
              >
                <KeyRound className="h-4 w-4" /> Ganti Kata Sandi
              </Link>
            </div>
          </form>
        </section>
      </div>
    </DashboardLayout>
  );
}
