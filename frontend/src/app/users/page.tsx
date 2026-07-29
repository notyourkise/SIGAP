'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';
import { Eye, KeyRound, Pencil, Plus, Search, Trash2, UserCog, X } from 'lucide-react';

interface ManagedUser {
  id: string;
  username: string;
  full_name: string;
  email: string;
  role: 'admin' | 'teknisi';
  created_at: string;
}

const initialForm = { username: '', password: '', full_name: '', email: '', role: 'teknisi' as 'admin' | 'teknisi' };

export default function UsersPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
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
      setError(err instanceof Error ? err.message : 'Pengguna tidak dapat dimuat');
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
      alert(err instanceof Error ? err.message : 'Detail pengguna gagal dimuat');
    }
  };

  const saveUser = async (event: FormEvent) => {
    event.preventDefault();
    try {
      if (modal === 'create') {
        await apiRequest('/auth/users', { method: 'POST', body: JSON.stringify(form) });
      } else if (modal === 'edit' && selected) {
        await apiRequest(`/users/${selected.id}`, {
          method: 'PUT',
          body: JSON.stringify({ full_name: form.full_name, email: form.email, role: form.role }),
        });
      }
      setModal(null);
      loadUsers();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Pengguna gagal disimpan');
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
      alert('Sandi pengguna berhasil diperbarui.');
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Sandi gagal diperbarui');
    }
  };

  const removeUser = async (item: ManagedUser) => {
    if (!confirm(`Hapus pengguna “${item.full_name}”?`)) return;
    try {
      await apiRequest(`/users/${item.id}`, { method: 'DELETE' });
      loadUsers();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Pengguna gagal dihapus');
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
          {loading ? <div className="py-20 text-center text-sm text-muted">Memuat pengguna…</div> :
            error ? <div className="py-20 text-center text-sm text-error">{error}</div> :
            users.length === 0 ? <div className="py-20 text-center text-sm text-muted">Belum ada pengguna.</div> : (
              <>
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[760px] text-left text-sm">
                    <thead className="border-b border-border bg-background/60 text-xs uppercase tracking-wider text-muted">
                      <tr><th className="px-5 py-4">Pengguna</th><th className="px-5 py-4">Email</th><th className="px-5 py-4">Role</th><th className="px-5 py-4 text-right">Aksi</th></tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {users.map((item) => (
                        <tr key={item.id} className="hover:bg-background/40">
                          <td className="px-5 py-4"><strong>{item.full_name}</strong><div className="mt-1 text-xs text-muted">@{item.username}</div></td>
                          <td className="px-5 py-4">{item.email}</td>
                          <td className="px-5 py-4"><span className={`rounded-full px-2.5 py-1 text-xs font-semibold capitalize ${item.role === 'admin' ? 'bg-warning/10 text-warning' : 'bg-success/10 text-success'}`}>{item.role}</span></td>
                          <td className="px-5 py-4"><div className="flex justify-end gap-2">
                            <Action title="Detail" onClick={() => openUser(item, 'detail')}><Eye /></Action>
                            <Action title="Ubah" onClick={() => openUser(item, 'edit')}><Pencil /></Action>
                            <Action title="Reset sandi" onClick={() => openUser(item, 'password')}><KeyRound /></Action>
                            <Action title="Hapus" onClick={() => removeUser(item)} danger><Trash2 /></Action>
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
            {modal === 'create' && <Field label="Password" type="password" value={form.password} onChange={(value) => setForm({ ...form, password: value })} minLength={6} />}
            <label className="block space-y-1.5 text-xs font-semibold text-muted">Role
              <select value={form.role} onChange={(event) => setForm({ ...form, role: event.target.value as 'admin' | 'teknisi' })} className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground">
                <option value="teknisi">Teknisi</option><option value="admin">Admin</option>
              </select>
            </label>
            <SubmitRow onCancel={() => setModal(null)} />
          </form>
        </Modal>
      )}

      {modal === 'detail' && selected && (
        <Modal title="Detail Pengguna" onClose={() => setModal(null)}>
          <div className="flex items-start gap-4">
            <div className="rounded-xl bg-primary/10 p-3 text-primary"><UserCog className="h-6 w-6" /></div>
            <dl className="grid flex-1 gap-4 sm:grid-cols-2">
              {Object.entries({ 'Nama lengkap': selected.full_name, Username: selected.username, Email: selected.email, Role: selected.role }).map(([label, value]) => <div key={label}><dt className="text-xs font-semibold uppercase text-muted">{label}</dt><dd className="mt-1 font-medium">{value}</dd></div>)}
            </dl>
          </div>
        </Modal>
      )}

      {modal === 'password' && selected && (
        <Modal title={`Reset Sandi — ${selected.full_name}`} onClose={() => setModal(null)}>
          <form onSubmit={resetPassword} className="space-y-4">
            <Field label="Sandi baru" type="password" value={newPassword} onChange={setNewPassword} minLength={6} />
            <SubmitRow onCancel={() => setModal(null)} label="Reset Sandi" />
          </form>
        </Modal>
      )}
    </DashboardLayout>
  );
}

function Action({ title, onClick, danger, children }: { title: string; onClick: () => void; danger?: boolean; children: React.ReactNode }) {
  return <button title={title} onClick={onClick} className={`rounded-lg border border-border p-2 [&>svg]:h-4 [&>svg]:w-4 ${danger ? 'hover:text-error' : 'hover:text-primary'}`}>{children}</button>;
}

function Field({ label, value, onChange, type = 'text', minLength }: { label: string; value: string; onChange: (value: string) => void; type?: string; minLength?: number }) {
  return <label className="block space-y-1.5 text-xs font-semibold text-muted">{label}<input required type={type} minLength={minLength} value={value} onChange={(event) => onChange(event.target.value)} className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground outline-none focus:border-primary" /></label>;
}

function SubmitRow({ onCancel, label = 'Simpan' }: { onCancel: () => void; label?: string }) {
  return <div className="flex justify-end gap-2 border-t border-border pt-4"><button type="button" onClick={onCancel} className="rounded-xl border border-border px-4 py-2.5 text-sm font-semibold">Batal</button><button className="rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-white">{label}</button></div>;
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return <div className="fixed inset-0 z-50 flex items-center justify-center p-4"><button aria-label="Tutup modal" className="absolute inset-0 bg-black/60" onClick={onClose} /><div className="relative z-10 w-full max-w-lg rounded-2xl border border-border bg-card p-6 shadow-2xl"><div className="mb-5 flex items-center justify-between"><h2 className="text-xl font-bold">{title}</h2><button onClick={onClose} className="rounded-lg border border-border p-2"><X className="h-4 w-4" /></button></div>{children}</div></div>;
}
