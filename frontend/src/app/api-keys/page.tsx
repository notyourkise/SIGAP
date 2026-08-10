'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import SortableHeader from '@/components/SortableHeader';
import { useAuth } from '@/context/AuthContext';
import { useSortableRows } from '@/hooks/useSortableRows';
import { apiRequest } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import {
  Plus,
  Trash2,
  KeyRound,
  AlertTriangle,
  Search,
  X,
  LoaderCircle,
  Copy,
  Check,
  Shield,
} from 'lucide-react';

const SCOPES = [
  { value: 'cctv:read', label: 'CCTV (baca)' },
  { value: 'maintenance:read', label: 'Pemeliharaan (baca)' },
  { value: 'health:read', label: 'Kesehatan layanan' },
];

const SCOPE_LABELS: Record<string, string> = {
  'cctv:read': 'CCTV',
  'maintenance:read': 'Pemeliharaan',
  'health:read': 'Kesehatan',
};

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  expires_at: string | null;
  last_used_at: string | null;
  revoked_at: string | null;
  created_at: string;
  user?: { full_name?: string };
}

interface APIKeysResponse {
  rows: APIKey[];
  pagination: { page: number; limit: number; total_rows: number; total_pages: number };
}

const formatDate = (iso: string | null) => {
  if (!iso) return 'Tidak pernah';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString('id-ID', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
};

export default function APIKeysPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
  const toast = useToast();
  const confirm = useConfirm();

  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({ page: 1, limit: 10, total_rows: 0, total_pages: 1 });
  const [search, setSearch] = useState('');

  // Modal create
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [formName, setFormName] = useState('');
  const [formScopes, setFormScopes] = useState<string[]>(['cctv:read']);
  const [formExpiresDays, setFormExpiresDays] = useState(0);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Hasil generate (key mentah tampil 1×)
  const [generated, setGenerated] = useState<{ api_key: string; key_prefix: string; name: string } | null>(null);
  const [copied, setCopied] = useState(false);

  const { sortedRows, sortKey, sortDirection, onSort } = useSortableRows<APIKey>(keys, {
    getValue: (row, key) => {
      switch (key) {
        case 'name':
          return row.name;
        case 'key_prefix':
          return row.key_prefix;
        case 'scopes':
          return (row.scopes || []).join(', ');
        case 'last_used_at':
          return row.last_used_at;
        case 'expires_at':
          return row.expires_at;
        default:
          return null;
      }
    },
    defaultKey: 'name',
    defaultDirection: 'asc',
  });

  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user]);

  const fetchKeys = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const params = new URLSearchParams({ page: String(page), limit: '10' });
      if (search) params.set('search', search);
      const response = await apiRequest<APIKeysResponse>(`/api-keys?${params}`);
      if (response.status === 'success' && response.data) {
        setKeys(response.data.rows || []);
        setPagination(response.data.pagination || { page: 1, limit: 10, total_rows: 0, total_pages: 1 });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Data API key belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [page, search]);

  useEffect(() => {
    if (user?.role !== 'admin') return;
    let cancelled = false;
    const load = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchKeys();
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [fetchKeys, user?.role]);

  const toggleScope = (scope: string) => {
    setFormScopes((prev) => (prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope]));
  };

  const handleGenerate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formName.trim()) {
      toast.error('Nama API key wajib diisi.');
      return;
    }
    if (formScopes.length === 0) {
      toast.error('Pilih minimal satu scope.');
      return;
    }
    setActionLoading('create');
    try {
      const response = await apiRequest<{ api_key: string; key_prefix: string; name: string }>('/api-keys', {
        method: 'POST',
        body: JSON.stringify({ name: formName, scopes: formScopes, expires_in_days: formExpiresDays }),
      });
      if (response.status === 'success' && response.data) {
        setGenerated(response.data);
        setIsCreateOpen(false);
        setFormName('');
        setFormScopes(['cctv:read']);
        setFormExpiresDays(0);
        fetchKeys();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'API key belum dapat dibuat.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleRevoke = async (key: APIKey) => {
    const ok = await confirm({
      title: 'Cabut API Key?',
      message: `Cabut API key "${key.name}" (${key.key_prefix}…)? Key tidak dapat digunakan lagi.`,
      confirmLabel: 'Cabut',
    });
    if (!ok) return;
    setActionLoading(`revoke-${key.id}`);
    try {
      const response = await apiRequest(`/api-keys/${key.id}/revoke`, { method: 'POST' });
      if (response.status === 'success') {
        toast.success(`API key "${key.name}" berhasil dicabut.`);
        fetchKeys();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'API key belum dapat dicabut.');
    } finally {
      setActionLoading(null);
    }
  };

  const copyKey = async () => {
    if (!generated) return;
    try {
      await navigator.clipboard.writeText(generated.api_key);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error('Gagal menyalin key.');
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">API Keys</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Kelola kredensial untuk integrasi eksternal. Key mentah hanya ditampilkan sekali saat pembuatan.
            </p>
          </div>
          <button
            type="button"
            onClick={() => setIsCreateOpen(true)}
            disabled={actionLoading !== null}
            className="flex min-h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.18)] transition-colors hover:bg-primary-hover disabled:opacity-60"
          >
            <Plus className="h-4 w-4" />
            Buat API Key
          </button>
        </div>

        {/* Hasil generate (key mentah tampil 1×) */}
        {generated && (
          <div role="alert" className="rounded-2xl border border-primary/30 bg-primary/5 p-5">
            <div className="flex items-start gap-3">
              <Shield className="mt-0.5 h-5 w-5 shrink-0 text-primary" />
              <div className="flex-1">
                <h2 className="font-semibold">API Key berhasil dibuat — simpan sekarang!</h2>
                <p className="mt-1 text-sm text-muted">
                  Key untuk &quot;{generated.name}&quot; tidak akan ditampilkan lagi. Salin dan simpan di tempat aman.
                </p>
                <div className="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
                  <code className="break-all rounded-lg border border-border bg-background px-3 py-2 font-mono text-xs text-primary">
                    {generated.api_key}
                  </code>
                  <button
                    type="button"
                    onClick={copyKey}
                    className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover"
                  >
                    {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                    {copied ? 'Tersalin' : 'Salin Key'}
                  </button>
                  <button
                    type="button"
                    onClick={() => setGenerated(null)}
                    className="shrink-0 rounded-lg border border-border px-3 py-2 text-xs font-semibold transition-colors hover:bg-background"
                  >
                    Tutup
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        <section aria-labelledby="api-keys-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex flex-col gap-3 border-b border-border px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 id="api-keys-heading" className="font-semibold">Daftar API Key</h2>
              <p className="mt-0.5 text-xs text-muted">{pagination.total_rows} key</p>
            </div>
            <div className="relative max-w-md">
              <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
              <input
                type="search"
                aria-label="Cari API key"
                placeholder="Cari nama key"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-xl border border-input-border bg-input py-2.5 pl-10 pr-4 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center gap-3 py-20" role="status">
              <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
              <span className="text-sm text-muted">Memuat API key…</span>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <AlertTriangle className="h-10 w-10 text-error" />
              <h3 className="mt-3 text-base font-semibold">Data API key belum dapat dimuat</h3>
              <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
              <button type="button" onClick={fetchKeys} className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover">Coba lagi</button>
            </div>
          ) : keys.length === 0 ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <KeyRound className="h-10 w-10 text-muted/45" />
              <h3 className="mt-3 text-base font-semibold">{search ? 'Tidak ada API key yang cocok' : 'Belum ada API key'}</h3>
              <p className="mt-1 max-w-sm text-xs text-muted">Gunakan &quot;Buat API Key&quot; untuk integrasi eksternal.</p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[880px] border-collapse text-left">
                  <thead>
                    <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                      <th className="px-5 py-3.5"><SortableHeader label="Nama" sortKey="name" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Key" sortKey="key_prefix" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Scope" sortKey="scopes" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Terakhir Dipakai" sortKey="last_used_at" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Kedaluwarsa" sortKey="expires_at" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5 text-right">Tindakan</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border text-sm">
                    {sortedRows.map((key) => (
                      <tr key={key.id} className="transition-colors hover:bg-background/40">
                        <td className="px-5 py-4">
                          <p className="font-semibold">{key.name}</p>
                          {key.user?.full_name && <p className="text-xs text-muted">oleh {key.user.full_name}</p>}
                        </td>
                        <td className="px-5 py-4">
                          <span className="font-mono text-xs font-semibold text-primary">{key.key_prefix}…</span>
                        </td>
                        <td className="px-5 py-4">
                          <div className="flex flex-wrap gap-1">
                            {(key.scopes || []).map((s) => (
                              <span key={s} className="rounded-full border border-border bg-background px-2 py-0.5 text-[11px] font-semibold">
                                {SCOPE_LABELS[s] || s}
                              </span>
                            ))}
                          </div>
                        </td>
                        <td className="px-5 py-4 text-xs text-muted">{formatDate(key.last_used_at)}</td>
                        <td className="px-5 py-4 text-xs text-muted">{formatDate(key.expires_at)}</td>
                        <td className="px-5 py-4 text-right">
                          {key.revoked_at ? (
                            <span className="rounded-full border border-error/25 bg-error/10 px-2.5 py-1 text-[11px] font-bold text-error">Dicabut</span>
                          ) : (
                            <button
                              type="button"
                              onClick={() => handleRevoke(key)}
                              disabled={actionLoading !== null}
                              className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 text-xs font-bold text-error transition-colors hover:bg-error/10 disabled:opacity-50"
                              title="Cabut API key"
                            >
                              {actionLoading === `revoke-${key.id}` ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                              Cabut
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Pagination page={pagination.page} limit={pagination.limit} totalRows={pagination.total_rows} totalPages={pagination.total_pages} onPageChange={setPage} itemLabel="API key" />
            </>
          )}
        </section>
      </div>

      {/* MODAL BUAT API KEY */}
      {isCreateOpen && (
        <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => { if (actionLoading !== 'create') setIsCreateOpen(false); }}></div>
          <div role="dialog" aria-modal="true" aria-labelledby="apikey-form-title" className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6">
            <div className="mb-6 flex items-start justify-between gap-4">
              <div>
                <h3 id="apikey-form-title" className="text-xl font-bold tracking-[-0.015em]">Buat API Key</h3>
                <p className="mt-1 text-sm text-muted">Beri nama dan pilih scope izin untuk integrasi eksternal.</p>
              </div>
              <button type="button" aria-label="Tutup" onClick={() => { if (actionLoading !== 'create') setIsCreateOpen(false); }} disabled={actionLoading === 'create'} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground disabled:opacity-50">
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleGenerate} className="space-y-4">
              <div>
                <label htmlFor="apikey-name" className="mb-1.5 block text-xs font-semibold text-muted">Nama *</label>
                <input id="apikey-name" autoFocus type="text" value={formName} onChange={(e) => setFormName(e.target.value)} placeholder="Contoh: Integrasi Dashboard OIKN" className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all" required />
              </div>

              <div>
                <span className="mb-1.5 block text-xs font-semibold text-muted">Scope izin *</span>
                <div className="space-y-2">
                  {SCOPES.map((s) => (
                    <label key={s.value} className="flex cursor-pointer items-center gap-3 rounded-xl border border-border px-3 py-2.5 hover:bg-background/40">
                      <input type="checkbox" checked={formScopes.includes(s.value)} onChange={() => toggleScope(s.value)} className="h-4 w-4 rounded border-input-border text-primary focus:ring-primary/30" />
                      <span className="text-sm font-semibold">{s.label}</span>
                      <code className="ml-auto font-mono text-[11px] text-muted">{s.value}</code>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label htmlFor="apikey-expire" className="mb-1.5 block text-xs font-semibold text-muted">Kedaluwarsa (hari)</label>
                <input id="apikey-expire" type="number" min={0} value={formExpiresDays} onChange={(e) => setFormExpiresDays(Number(e.target.value) || 0)} className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all" />
                <p className="mt-1 text-xs text-muted">0 = tidak pernah kedaluwarsa.</p>
              </div>

              <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
                <button type="button" onClick={() => { if (actionLoading !== 'create') setIsCreateOpen(false); }} disabled={actionLoading === 'create'} className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background disabled:opacity-50">Batal</button>
                <button type="submit" disabled={actionLoading === 'create'} className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover disabled:opacity-65">
                  {actionLoading === 'create' && <LoaderCircle className="h-4 w-4 animate-spin" />}
                  {actionLoading === 'create' ? 'Membuat…' : 'Buat API Key'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}
