'use client';

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import {
  AtSign,
  BadgeCheck,
  Calendar,
  Camera,
  KeyRound,
  LoaderCircle,
  Mail,
  Save,
  ShieldAlert,
  Trash2,
  UserRound,
  MailCheck,
} from 'lucide-react';
import DashboardLayout from '@/components/DashboardLayout';
import AuthenticatedImage from '@/components/AuthenticatedImage';
import PushNotificationOptIn from '@/components/PushNotificationOptIn';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';

const MAX_PHOTO_SIZE = 2 * 1024 * 1024; // 2MB — sama dengan batas backend

interface ProfileResponse {
  id: string;
  username: string;
  full_name: string;
  email: string;
  role: 'admin' | 'teknisi';
  profile_photo_url: string;
  email_notifications?: boolean;
  email_daily_digest?: boolean;
  email_overdue?: boolean;
  email_stock_low?: boolean;
}

export default function ProfilePage() {
  const { user, updateUser } = useAuth();
  const [fullName, setFullName] = useState(user?.full_name ?? '');
  const [photoUrl, setPhotoUrl] = useState(user?.profile_photo_url ?? '');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploadingPhoto, setUploadingPhoto] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Preferensi notifikasi email (Task 11.3 + Task 13.7 granular)
  const [emailNotifications, setEmailNotifications] = useState<boolean>(
    user?.email_notifications ?? true
  );
  const [emailDailyDigest, setEmailDailyDigest] = useState<boolean>(
    user?.email_daily_digest ?? true
  );
  const [emailOverdue, setEmailOverdue] = useState<boolean>(
    user?.email_overdue ?? true
  );
  const [emailStockLow, setEmailStockLow] = useState<boolean>(
    user?.email_stock_low ?? true
  );
  const [savingEmailPref, setSavingEmailPref] = useState(false);

  const saveProfile = async (fullNameValue: string, photoUrlValue: string) => {
    const response = await apiRequest<ProfileResponse>('/auth/profile', {
      method: 'PUT',
      body: JSON.stringify({
        full_name: fullNameValue.trim(),
        profile_photo_url: photoUrlValue,
      }),
    });
    if (response.status === 'success' && response.data) {
      updateUser(response.data);
      return true;
    }
    throw new Error(response.message || 'Profil belum dapat diperbarui. Periksa kembali isian Anda.');
  };

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
      await saveProfile(fullName, photoUrl);
      setSuccess('Profil berhasil diperbarui.');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Layanan SIGAP belum dapat dihubungi. Periksa koneksi lalu coba lagi.');
    } finally {
      setLoading(false);
    }
  };

  const handlePhotoChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;

    setError('');
    setSuccess('');

    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
      setError('Format foto tidak didukung. Gunakan JPEG, PNG, atau WebP.');
      return;
    }
    if (file.size > MAX_PHOTO_SIZE) {
      setError('Ukuran foto terlalu besar. Maksimal 2MB.');
      return;
    }

    setUploadingPhoto(true);
    try {
      const formData = new FormData();
      formData.append('file', file);
      const upload = await apiRequest<{ url: string }>('/upload', { method: 'POST', body: formData });
      if (upload.status !== 'success' || !upload.data?.url) {
        throw new Error(upload.message || 'Gagal mengunggah foto.');
      }
      await saveProfile(fullName, upload.data.url);
      setPhotoUrl(upload.data.url);
      setSuccess('Foto profil berhasil diperbarui.');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal mengunggah foto profil.');
    } finally {
      setUploadingPhoto(false);
    }
  };

  const handleRemovePhoto = async () => {
    setError('');
    setSuccess('');
    setUploadingPhoto(true);
    try {
      await saveProfile(fullName, '');
      setPhotoUrl('');
      setSuccess('Foto profil dihapus.');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal menghapus foto profil.');
    } finally {
      setUploadingPhoto(false);
    }
  };

  // Kirim preferensi email ke backend (auto-save). payload berisi field yang berubah.
  const persistEmailPrefs = async (payload: Record<string, boolean>) => {
    setSavingEmailPref(true);
    try {
      const response = await apiRequest<ProfileResponse>('/profile/email-preferences', {
        method: 'PUT',
        body: JSON.stringify(payload),
      });
      if (response.status === 'success' && response.data) {
        setEmailNotifications(response.data.email_notifications ?? emailNotifications);
        setEmailDailyDigest(response.data.email_daily_digest ?? emailDailyDigest);
        setEmailOverdue(response.data.email_overdue ?? emailOverdue);
        setEmailStockLow(response.data.email_stock_low ?? emailStockLow);
        if (user) {
          updateUser({ ...user, ...response.data });
        }
        setSuccess('Preferensi email berhasil disimpan.');
      } else {
        throw new Error(response.message || 'Preferensi email belum dapat disimpan.');
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Preferensi email belum dapat disimpan. Coba lagi.');
      // Re-sync dari user (rollback).
      setEmailNotifications(user?.email_notifications ?? true);
      setEmailDailyDigest(user?.email_daily_digest ?? true);
      setEmailOverdue(user?.email_overdue ?? true);
      setEmailStockLow(user?.email_stock_low ?? true);
    } finally {
      setSavingEmailPref(false);
    }
  };

  // Debounce ref untuk auto-save.
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Master toggle: bila dimatikan, kirim email_notifications=false (backend juga
  // mematikan sub-prefs). Bila dinyalakan, kirim email_notifications=true.
  const handleToggleMasterEmail = (checked: boolean) => {
    setEmailNotifications(checked);
    if (!checked) {
      setEmailDailyDigest(false);
      setEmailOverdue(false);
      setEmailStockLow(false);
    }
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      persistEmailPrefs({ email_notifications: checked });
    }, 500);
  };

  // Sub-toggle granular per-tipe.
  const handleToggleSubEmail = (
    key: 'email_daily_digest' | 'email_overdue' | 'email_stock_low',
    checked: boolean
  ) => {
    if (key === 'email_daily_digest') setEmailDailyDigest(checked);
    else if (key === 'email_overdue') setEmailOverdue(checked);
    else setEmailStockLow(checked);
    // Mematikan sub-pref juga mematikan master bila semua nonaktif; backend menangani sinkronisasi.
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      persistEmailPrefs({ [key]: checked });
    }, 500);
  };

  // Bersihkan timer debounce saat unmount.
  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

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
            <div className="flex flex-wrap items-center gap-4 border-b border-border px-5 py-5">
              <div className="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-2xl bg-accent/20 text-[#775800] dark:text-accent">
                {photoUrl ? (
                  <AuthenticatedImage imageUrl={photoUrl} alt={user.full_name} className="h-full w-full rounded-2xl object-cover" />
                ) : (
                  <UserRound className="h-8 w-8" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <h2 id="profile-details-heading" className="truncate text-lg font-bold">{user.full_name}</h2>
                <span className={`mt-1 inline-block rounded-md border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider ${
                  user.role === 'admin'
                    ? 'bg-accent/20 text-[#775800] border border-accent/30 dark:text-accent'
                    : 'bg-success/10 text-success border border-success/20'
                }`}>
                  {user.role}
                </span>
              </div>
              <div className="flex flex-col gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  onChange={handlePhotoChange}
                  className="hidden"
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={uploadingPhoto || loading}
                  className="flex items-center justify-center gap-2 rounded-xl bg-primary px-3.5 py-2 text-xs font-semibold text-white shadow-md shadow-primary/20 transition-colors hover:bg-primary-hover disabled:cursor-wait disabled:opacity-60"
                >
                  {uploadingPhoto ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Camera className="h-3.5 w-3.5" />}
                  {uploadingPhoto ? 'Mengunggah…' : photoUrl ? 'Ubah Foto' : 'Tambah Foto'}
                </button>
                {photoUrl && (
                  <button
                    type="button"
                    onClick={handleRemovePhoto}
                    disabled={uploadingPhoto || loading}
                    className="flex items-center justify-center gap-2 rounded-xl border border-error/30 bg-error/5 px-3.5 py-2 text-xs font-semibold text-red-600 transition-colors hover:bg-error/10 disabled:cursor-wait disabled:opacity-60 dark:text-red-400"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                    Hapus Foto
                  </button>
                )}
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
            <p className="mt-0.5 text-xs text-muted">Nama ini ditampilkan pada badge profil dan aktivitas sistem.</p>
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

        {/* Notifikasi push browser (Task 8.3) */}
        <section aria-labelledby="push-settings-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="border-b border-border px-5 py-4">
            <h2 id="push-settings-heading" className="text-base font-bold">Notifikasi Push</h2>
            <p className="mt-0.5 text-xs text-muted">
              Terima notifikasi OS saat ada tugas baru, eskalasi, atau pembaruan penting.
            </p>
          </div>
          <div className="p-5">
            <PushNotificationOptIn />
          </div>
        </section>

        {/* Preferensi email (Task 11.3 + Task 13.7 granular) */}
        <section aria-labelledby="email-settings-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="border-b border-border px-5 py-4">
            <h2 id="email-settings-heading" className="text-base font-bold">Notifikasi Email</h2>
            <p className="mt-0.5 text-xs text-muted">
              Pilih jenis email yang ingin Anda terima. Perubahan tersimpan otomatis.
            </p>
          </div>
          <div className="space-y-4 p-5">
            {/* Master toggle */}
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-start gap-3">
                <MailCheck className="mt-0.5 h-5 w-5 shrink-0 text-primary" />
                <div>
                  <p className="text-sm font-medium">
                    Aktifkan notifikasi email {user?.email ? `(${user.email})` : ''}
                  </p>
                  <p className="mt-1 max-w-lg text-xs text-muted">
                    {emailNotifications
                      ? 'Aktif — Anda akan menerima email otomatis dari SIGAP.'
                      : 'Nonaktif — Anda hanya menerima notifikasi in-app dan push.'}
                  </p>
                </div>
              </div>
              <ToggleSwitch checked={emailNotifications} disabled={savingEmailPref || !user?.email} onChange={handleToggleMasterEmail} label="Aktifkan atau nonaktifkan notifikasi email" />
            </div>

            {/* Sub-toggle granular */}
            <div className={`space-y-3 rounded-xl border border-border bg-background/40 p-4 transition-opacity ${emailNotifications ? '' : 'opacity-60'}`}>
              <EmailPrefRow
                icon="📅"
                title="Ringkasan Harian"
                description="Terima laporan operasional setiap pagi pukul 06:00 WITA."
                checked={emailDailyDigest}
                disabled={savingEmailPref || !emailNotifications || !user?.email}
                onChange={(v) => handleToggleSubEmail('email_daily_digest', v)}
              />
              <EmailPrefRow
                icon="⚠️"
                title="Eskalasi Overdue"
                description="Notif saat tugas melewati batas waktu penyelesaian."
                checked={emailOverdue}
                disabled={savingEmailPref || !emailNotifications || !user?.email}
                onChange={(v) => handleToggleSubEmail('email_overdue', v)}
              />
              <EmailPrefRow
                icon="📦"
                title="Stok Suku Cadang Rendah"
                description="Alert saat stok suku cadang di bawah minimum."
                checked={emailStockLow}
                disabled={savingEmailPref || !emailNotifications || !user?.email}
                onChange={(v) => handleToggleSubEmail('email_stock_low', v)}
              />
            </div>

            {!user?.email && (
              <p className="rounded-xl border border-warning/25 bg-warning/10 px-4 py-3 text-xs text-warning">
                Belum ada email terdaftar pada akun Anda. Hubungi admin untuk melengkapi alamat email.
              </p>
            )}
          </div>
        </section>
      </div>
    </DashboardLayout>
  );
}

// Toggle switch reusable (role=switch) untuk preferensi email.
function ToggleSwitch({
  checked,
  onChange,
  disabled,
  label,
}: {
  checked: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
  label: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-not-allowed disabled:opacity-60 ${
        checked ? 'bg-primary' : 'bg-border'
      }`}
      aria-label={label}
    >
      <span
        className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
          checked ? 'translate-x-6' : 'translate-x-1'
        }`}
      />
    </button>
  );
}

// Baris preferensi email granular (ikon + judul + deskripsi + toggle).
function EmailPrefRow({
  icon,
  title,
  description,
  checked,
  onChange,
  disabled,
}: {
  icon: string;
  title: string;
  description: string;
  checked: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="flex items-start gap-3">
        <span className="mt-0.5 h-5 w-5 shrink-0 text-lg leading-none" aria-hidden="true">{icon}</span>
        <div>
          <p className="text-sm font-medium">{title}</p>
          <p className="mt-0.5 max-w-md text-xs text-muted">{description}</p>
        </div>
      </div>
      <ToggleSwitch checked={checked} onChange={onChange} disabled={disabled} label={`Aktifkan atau nonaktifkan ${title}`} />
    </div>
  );
}
