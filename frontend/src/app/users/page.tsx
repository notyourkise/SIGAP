'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import SortableHeader from '@/components/SortableHeader';
import { useAuth } from '@/context/AuthContext';
import { useSortableRows } from '@/hooks/useSortableRows';
import { apiRequest } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { Eye, KeyRound, Mail, CalendarDays, Pencil, Plus, Search, Trash2, X } from 'lucide-react';

interface ManagedUser {
  id: string;
  username: string;
  full_name: string;
  email: string;
  role: 'admin' | 'teknisi';
  created_at: string;
}

const initialForm = { username: '', password: '', full_name: '', email: '', role: 'teknisi' as 'admin' | 'teknisi' };

// Inisial nama untuk avatar (dua kata pertama)
const getInitials = (name: string) => {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[1][0]).toUpperCase();
};

const formatUserDate = (iso: string) => {
  if (!iso) return '-';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString('id-ID', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

export default function UsersPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
  const toast = useToast();
  const confirm = useConfirm();
  const [users, setUsers] = useState<ManagedUser[]>([]);
  const [pagination, setPagination] = useState({ page: 1, limit: 10, total_rows: 0, total_pages: 1 });
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [modal, setModal] = useState<'create' | 'edit' | 'detail' | 'password' | null>(null);
  const [selected, setSelected] = useState<ManagedUser | null>(null);
  const [form, setForm] = useState(initialForm);
  const [newPassword, setNewPassword] = useState('');

  const { sortedRows, sortKey, sortDirection, onSort } = useSortableRows<ManagedUser>(users, {
    getValue: (row, key) => {
      switch (key) {
        case 'full_name':
          return row.full_name;
        case 'email':
          return row.email;
        case 'role':
          return row.role;
        default:
          return null;
      }
    },
    defaultKey: 'full_name',
    defaultDirection: 'asc',
  });

  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user]);

  const loadUsers = useCallback(async () => {
    if (user?.role !== 'admin') return;
    setLoading(true);
    setError('');
    const params = new URLSearchParams({ page: String(page), limit: '10' });
    if (search) params.set('search', search);
    try {
      const response = await apiRequest<{ rows: ManagedUser[]; pagination: typeof pagination }>(`/users?${params}`);
      setUsers(response.data?.rows || []);
      if (response.data?.pagination) setPagination(response.data.pagination);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Daftar pengguna belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [page, search, user?.role]);

  useEffect(() => {
    const timer = setTimeout(loadUsers, 250);
    return () => clearTimeout(timer);
  }, [loadUsers]);

  const openCreate = () => {
    setSelected(null);
    setForm(initialForm);
    setModal('create');
  };

  const openUser = async (item: ManagedUser, mode: 'edit' | 'detail' | 'password') => {
    try {
      const response = await apiRequest<ManagedUser>(`/users/${item.id}`);
      const target = response.data || item;
      setSelected(target);
      setForm({ username: target.username, password: '', full_name: target.full_name, email: target.email, role: target.role });
      setNewPassword('');
      setModal(mode);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Detail pengguna belum dapat dimuat. Coba lagi.');
    }
  };

  const saveUser = async (event: FormEvent) => {
    event.preventDefault();
    try {
      if (modal === 'create') {
        await apiRequest('/auth/users', { method: 'POST', body: JSON.stringify(form) });
        toast.success(`Pengguna "${form.full_name}" berhasil ditambahkan.`);
      } else if (modal === 'edit' && selected) {
        await apiRequest(`/users/${selected.id}`, {
          method: 'PUT',
          body: JSON.stringify({ full_name: form.full_name, email: form.email, role: form.role }),
        });
        toast.success(`Data pengguna "${form.full_name}" berhasil diperbarui.`);
      }
      setModal(null);
      loadUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Data pengguna belum dapat disimpan. Periksa isian lalu coba lagi.');
    }
  };

  const resetPassword = async (event: FormEvent) => {
    event.preventDefault();
    if (!selected) return;
    try {
      await apiRequest(`/users/${selected.id}/reset-password`, {
        method: 'POST',
        body: JSON.stringify({ new_password: newPassword }),
      });
      setModal(null);
      toast.success('Kata sandi pengguna berhasil diperbarui.');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Kata sandi belum dapat diperbarui. Coba lagi.');
    }
  };

  const removeUser = async (item: ManagedUser) => {
    const ok = await confirm({
      title: 'Hapus Pengguna?',
      message: `Hapus pengguna “${item.full_name}”? Pengguna ini tidak akan dapat mengakses SIGAP.`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    try {
      await apiRequest(`/users/${item.id}`, { method: 'DELETE' });
      toast.success(`Pengguna "${item.full_name}" berhasil dihapus.`);
      loadUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Pengguna belum dapat dihapus. Coba lagi.');
    }
  };

  const handleDeleteFromDetail = async () => {
    if (!selected) return;
    const ok = await confirm({
      title: 'Hapus Pengguna?',
      message: `Hapus pengguna “${selected.full_name}”? Pengguna ini tidak akan dapat mengakses SIGAP.`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    try {
      await apiRequest(`/users/${selected.id}`, { method: 'DELETE' });
      setModal(null);
      toast.success(`Pengguna "${selected.full_name}" berhasil dihapus.`);
      loadUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Pengguna belum dapat dihapus. Coba lagi.');
    }
  };

  if (!authLoading && user?.role !== 'admin') return null;

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Manajemen Pengguna</h1>
            <p className="mt-1 text-sm text-muted">Kelola akun admin dan teknisi sesuai kewenangannya.</p>
          </div>
          <button onClick={openCreate} className="inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-white">
            <Plus className="h-4 w-4" /> Tambah Pengguna
          </button>
        </div>

        <label className="flex max-w-xl items-center gap-2 rounded-xl border border-input-border bg-input px-3">
          <Search className="h-4 w-4 text-muted" />
          <input value={search} onChange={(event) => { setSearch(event.target.value); setPage(1); }} placeholder="Cari nama, username, atau email" className="w-full bg-transparent py-2.5 text-sm outline-none" />
        </label>

        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          {loading ? <div className="py-20 text-center text-sm text-muted" role="status">Memuat daftar pengguna…</div> :
            error ? <div className="py-16 text-center text-sm text-error"><p>{error}</p><button type="button" onClick={loadUsers} className="mt-4 rounded-xl border border-error/30 px-4 py-2 font-semibold">Coba lagi</button></div> :
            users.length === 0 ? <div className="py-20 text-center text-sm text-muted">{search ? 'Tidak ada pengguna yang cocok. Ubah kata kunci pencarian.' : 'Belum ada pengguna. Tambahkan pengguna untuk memberi akses ke SIGAP.'}</div> : (
              <>
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[760px] text-left text-sm">
                    <thead className="border-b border-border bg-background/60 text-xs uppercase tracking-wider text-muted">
                      <tr>
                        <th className="px-5 py-4"><SortableHeader label="Pengguna" sortKey="full_name" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                        <th className="px-5 py-4"><SortableHeader label="Email" sortKey="email" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                        <th className="px-5 py-4"><SortableHeader label="Peran" sortKey="role" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                        <th className="px-5 py-4 text-right">Tindakan</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {sortedRows.map((item) => (
                        <tr key={item.id} className="hover:bg-background/40">
                          <td className="px-5 py-4"><strong>{item.full_name}</strong><div className="mt-1 text-xs text-muted">@{item.username}</div></td>
                          <td className="px-5 py-4">{item.email}</td>
                          <td className="px-5 py-4"><span className={`rounded-full px-2.5 py-1 text-xs font-semibold capitalize ${item.role === 'admin' ? 'bg-warning/10 text-warning' : 'bg-success/10 text-success'}`}>{item.role}</span></td>
                          <td className="px-5 py-4"><div className="flex justify-end gap-2">
                            <Action title={`Lihat detail ${item.full_name}`} onClick={() => openUser(item, 'detail')}><Eye /></Action>
                            <Action title={`Ubah pengguna ${item.full_name}`} onClick={() => openUser(item, 'edit')}><Pencil /></Action>
                            <Action title={`Atur ulang kata sandi ${item.full_name}`} onClick={() => openUser(item, 'password')}><KeyRound /></Action>
                            <Action title={`Hapus pengguna ${item.full_name}`} onClick={() => removeUser(item)} danger><Trash2 /></Action>
                          </div></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <Pagination page={pagination.page} limit={pagination.limit} totalRows={pagination.total_rows} totalPages={pagination.total_pages} onPageChange={setPage} itemLabel="pengguna" />
              </>
            )}
        </div>
      </div>

      {(modal === 'create' || modal === 'edit') && (
        <Modal title={modal === 'create' ? 'Tambah Pengguna' : 'Ubah Pengguna'} onClose={() => setModal(null)}>
          <form onSubmit={saveUser} className="space-y-4">
            {modal === 'create' && <Field label="Username" value={form.username} onChange={(value) => setForm({ ...form, username: value })} minLength={4} />}
            <Field label="Nama lengkap" value={form.full_name} onChange={(value) => setForm({ ...form, full_name: value })} minLength={3} />
            <Field label="Email" type="email" value={form.email} onChange={(value) => setForm({ ...form, email: value })} />
            {modal === 'create' && <Field label="Kata sandi" type="password" value={form.password} onChange={(value) => setForm({ ...form, password: value })} minLength={8} />}
            <label className="block space-y-1.5 text-xs font-semibold text-muted">Peran
              <select value={form.role} onChange={(event) => setForm({ ...form, role: event.target.value as 'admin' | 'teknisi' })} className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground">
                <option value="teknisi">Teknisi</option><option value="admin">Admin</option>
              </select>
            </label>
            <SubmitRow onCancel={() => setModal(null)} label={modal === 'create' ? 'Tambah Pengguna' : 'Simpan Perubahan'} />
          </form>
        </Modal>
      )}

      {modal === 'detail' && selected && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <button type="button" aria-label="Tutup modal detail" className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setModal(null)} />

          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="user-detail-name"
            className="relative z-10 w-full max-w-lg overflow-hidden rounded-2xl border border-border bg-card shadow-2xl"
          >
            {/* Header identitas */}
            <div className="flex items-start gap-4 border-b border-border p-6">
              <div className={`flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl text-lg font-bold ${selected.role === 'admin' ? 'bg-warning/10 text-warning' : 'bg-primary/10 text-primary'}`}>
                {getInitials(selected.full_name)}
              </div>
              <div className="min-w-0 flex-1 pt-0.5">
                <div className="flex items-center gap-2">
                  <h3 id="user-detail-name" className="truncate text-xl font-bold tracking-[-0.015em]">{selected.full_name}</h3>
                  <span className={`inline-flex shrink-0 rounded-full border px-2.5 py-0.5 text-xs font-semibold capitalize ${selected.role === 'admin' ? 'border-warning/25 bg-warning/10 text-warning' : 'border-success/25 bg-success/10 text-success'}`}>
                    {selected.role}
                  </span>
                </div>
                <p className="mt-1 truncate font-mono text-xs font-semibold text-muted">@{selected.username}</p>
              </div>
              <button
                type="button"
                aria-label="Tutup detail pengguna"
                onClick={() => setModal(null)}
                className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Rincian akun */}
            <div className="px-6 py-5">
              <dl className="grid overflow-hidden rounded-xl border border-border bg-background text-sm sm:grid-cols-2">
                <div className="flex items-start gap-3 border-b border-border px-4 py-3.5 sm:border-b-0 sm:border-r">
                  <Mail className="mt-0.5 h-4 w-4 shrink-0 text-muted" />
                  <div className="min-w-0">
                    <dt className="text-xs font-semibold text-muted">Email</dt>
                    <dd className="mt-0.5 break-all font-medium">{selected.email || '-'}</dd>
                  </div>
                </div>
                <div className="flex items-start gap-3 px-4 py-3.5">
                  <CalendarDays className="mt-0.5 h-4 w-4 shrink-0 text-muted" />
                  <div className="min-w-0">
                    <dt className="text-xs font-semibold text-muted">Akun dibuat</dt>
                    <dd className="mt-0.5 font-medium">{formatUserDate(selected.created_at)}</dd>
                  </div>
                </div>
              </dl>
            </div>

            {/* Aksi cepat */}
            <div className="flex flex-col-reverse gap-2 border-t border-border px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
              <button
                type="button"
                onClick={handleDeleteFromDetail}
                className="inline-flex items-center justify-center gap-2 rounded-xl border border-error/30 px-4 py-2.5 text-sm font-semibold text-error transition-colors hover:bg-error/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error/40"
              >
                <Trash2 className="h-4 w-4" />
                Hapus pengguna
              </button>
              <div className="flex flex-col gap-2 sm:flex-row">
                <button
                  type="button"
                  onClick={() => setModal('edit')}
                  className="inline-flex items-center justify-center gap-2 rounded-xl border border-border px-4 py-2.5 text-sm font-semibold transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  <Pencil className="h-4 w-4" />
                  Ubah data
                </button>
                <button
                  type="button"
                  onClick={() => setModal('password')}
                  className="inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
                >
                  <KeyRound className="h-4 w-4" />
                  Atur ulang kata sandi
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {modal === 'password' && selected && (
        <Modal title={`Atur Ulang Kata Sandi — ${selected.full_name}`} onClose={() => setModal(null)}>
          <form onSubmit={resetPassword} className="space-y-4">
            <Field label="Kata sandi baru" type="password" value={newPassword} onChange={setNewPassword} minLength={8} />
            <SubmitRow onCancel={() => setModal(null)} label="Perbarui Kata Sandi" />
          </form>
        </Modal>
      )}
    </DashboardLayout>
  );
}

function Action({ title, onClick, danger, children }: { title: string; onClick: () => void; danger?: boolean; children: React.ReactNode }) {
  return <button type="button" title={title} aria-label={title} onClick={onClick} className={`rounded-lg border border-border p-2 [&>svg]:h-4 [&>svg]:w-4 ${danger ? 'hover:text-error' : 'hover:text-primary'}`}>{children}</button>;
}

function Field({ label, value, onChange, type = 'text', minLength }: { label: string; value: string; onChange: (value: string) => void; type?: string; minLength?: number }) {
  return <label className="block space-y-1.5 text-xs font-semibold text-muted">{label}<input required type={type} minLength={minLength} value={value} onChange={(event) => onChange(event.target.value)} className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground outline-none focus:border-primary" /></label>;
}

function SubmitRow({ onCancel, label = 'Simpan' }: { onCancel: () => void; label?: string }) {
  return <div className="flex justify-end gap-2 border-t border-border pt-4"><button type="button" onClick={onCancel} className="rounded-xl border border-border px-4 py-2.5 text-sm font-semibold">Batal</button><button className="rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-white">{label}</button></div>;
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return <div className="fixed inset-0 z-50 flex items-center justify-center p-4"><button type="button" aria-label="Tutup modal" className="absolute inset-0 bg-black/60" onClick={onClose} /><div className="relative z-10 w-full max-w-lg rounded-2xl border border-border bg-card p-6 shadow-2xl"><div className="mb-5 flex items-center justify-between"><h2 className="text-xl font-bold">{title}</h2><button type="button" aria-label={`Tutup ${title}`} onClick={onClose} className="rounded-lg border border-border p-2"><X className="h-4 w-4" /></button></div>{children}</div></div>;
}
