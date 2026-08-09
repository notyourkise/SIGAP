'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import {
  Plus,
  Trash2,
  Webhook,
  AlertTriangle,
  X,
  LoaderCircle,
  Send,
  Pencil,
  Eye,
  Zap,
  CheckCircle2,
  XCircle,
  Clock,
} from 'lucide-react';

const EVENTS = [
  { value: 'cctv.status_changed', label: 'Status CCTV berubah' },
  { value: 'maintenance.completed', label: 'Pemeliharaan selesai' },
  { value: 'maintenance.overdue', label: 'Pemeliharaan terlambat' },
  { value: 'health_score.changed', label: 'Skor kesehatan berubah' },
  { value: 'stream.down', label: 'Stream down' },
];

const EVENT_LABELS: Record<string, string> = {
  'cctv.status_changed': 'Status CCTV',
  'maintenance.completed': 'Pemeliharaan selesai',
  'maintenance.overdue': 'Pemeliharaan terlambat',
  'health_score.changed': 'Skor kesehatan',
  'stream.down': 'Stream down',
};

interface WebhookEndpoint {
  id: string;
  name: string;
  url: string;
  events: string[];
  is_active: boolean;
  created_at: string;
  user?: { full_name?: string };
}

interface Delivery {
  id: string;
  event: string;
  payload: string;
  response_code: number;
  response_body: string;
  latency: number;
  status: string;
  attempt: number;
  created_at: string;
}

interface WebhooksResponse {
  rows: WebhookEndpoint[];
  pagination: { page: number; limit: number; total_rows: number; total_pages: number };
}

const formatDate = (iso: string) => {
  if (!iso) return '-';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
};

const STATUS_BADGE: Record<string, { label: string; className: string }> = {
  success: { label: 'Sukses', className: 'border-success/25 bg-success/10 text-success' },
  failed: { label: 'Gagal', className: 'border-error/25 bg-error/10 text-error' },
  pending: { label: 'Pending', className: 'border-warning/25 bg-warning/10 text-warning' },
};

export default function WebhooksPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
  const toast = useToast();
  const confirm = useConfirm();

  const [endpoints, setEndpoints] = useState<WebhookEndpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({ page: 1, limit: 10, total_rows: 0, total_pages: 1 });
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Modal form
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [current, setCurrent] = useState<WebhookEndpoint | null>(null);
  const [form, setForm] = useState({ name: '', url: '', events: [] as string[], secret: '' });
  const [saveLoading, setSaveLoading] = useState(false);

  // Detail + deliveries
  const [detail, setDetail] = useState<{ endpoint: WebhookEndpoint; deliveries: Delivery[] } | null>(null);

  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user]);

  const fetchWebhooks = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const response = await apiRequest<WebhooksResponse>(`/webhooks?page=${page}&limit=10`);
      if (response.status === 'success' && response.data) {
        setEndpoints(response.data.rows || []);
        setPagination(response.data.pagination || { page: 1, limit: 10, total_rows: 0, total_pages: 1 });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Data webhook belum dapat dimuat.');
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    if (user?.role !== 'admin') return;
    let cancelled = false;
    const load = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchWebhooks();
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [fetchWebhooks, user?.role]);

  const toggleEvent = (event: string) => {
    setForm((prev) => ({ ...prev, events: prev.events.includes(event) ? prev.events.filter((e) => e !== event) : [...prev.events, event] }));
  };

  const openCreate = () => {
    setCurrent(null);
    setForm({ name: '', url: '', events: [], secret: '' });
    setIsFormOpen(true);
  };

  const openEdit = (ep: WebhookEndpoint) => {
    setCurrent(ep);
    setForm({ name: ep.name, url: ep.url, events: ep.events || [], secret: '' });
    setIsFormOpen(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim() || !form.url.trim()) {
      toast.error('Nama dan URL wajib diisi.');
      return;
    }
    if (form.events.length === 0) {
      toast.error('Pilih minimal satu event.');
      return;
    }
    if (!current && form.secret.length < 8) {
      toast.error('Secret minimal 8 karakter.');
      return;
    }
    setSaveLoading(true);
    try {
      const payload: Record<string, unknown> = { name: form.name, url: form.url, events: form.events };
      if (form.secret) payload.secret = form.secret;
      const endpoint = current ? `/webhooks/${current.id}` : '/webhooks';
      const method = current ? 'PUT' : 'POST';
      const response = await apiRequest<WebhookEndpoint>(endpoint, { method, body: JSON.stringify(payload) });
      if (response.status === 'success') {
        setIsFormOpen(false);
        toast.success(current ? `Webhook "${form.name}" diperbarui.` : `Webhook "${form.name}" dibuat.`);
        fetchWebhooks();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Webhook belum dapat disimpan.');
    } finally {
      setSaveLoading(false);
    }
  };

  const handleDelete = async (ep: WebhookEndpoint) => {
    const ok = await confirm({
      title: 'Hapus Webhook?',
      message: `Hapus endpoint webhook "${ep.name}"?`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    setActionLoading(`delete-${ep.id}`);
    try {
      const response = await apiRequest(`/webhooks/${ep.id}`, { method: 'DELETE' });
      if (response.status === 'success') {
        toast.success(`Webhook "${ep.name}" dihapus.`);
        fetchWebhooks();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Webhook belum dapat dihapus.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleToggle = async (ep: WebhookEndpoint) => {
    setActionLoading(`toggle-${ep.id}`);
    try {
      const response = await apiRequest<WebhookEndpoint>(`/webhooks/${ep.id}`, {
        method: 'PUT',
        body: JSON.stringify({ is_active: !ep.is_active }),
      });
      if (response.status === 'success') {
        toast.success(`Webhook "${ep.name}" ${ep.is_active ? 'dinonaktifkan' : 'diaktifkan'}.`);
        fetchWebhooks();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal mengubah status webhook.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleTest = async (ep: WebhookEndpoint) => {
    setActionLoading(`test-${ep.id}`);
    try {
      const response = await apiRequest<{ response_code: number; latency_ms: number; response_body?: string; error?: string }>(`/webhooks/${ep.id}/test`, { method: 'POST' });
      if (response.status === 'success') {
        toast.success(`Test terkirim — HTTP ${response.data?.response_code ?? '-'} (${response.data?.latency_ms ?? 0} ms)`);
      } else {
        toast.error(response.data?.error || 'Test webhook gagal.');
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Test webhook gagal.');
    } finally {
      setActionLoading(null);
    }
  };

  const openDetail = async (ep: WebhookEndpoint) => {
    try {
      const response = await apiRequest<{ endpoint: WebhookEndpoint; deliveries: Delivery[] }>(`/webhooks/${ep.id}`);
      if (response.status === 'success' && response.data) {
        setDetail(response.data);
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Detail webhook belum dapat dimuat.');
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Webhooks</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Kirim event perubahan (status CCTV, pemeliharaan, skor kesehatan, stream) ke sistem eksternal.
            </p>
          </div>
          <button type="button" onClick={openCreate} disabled={actionLoading !== null} className="flex min-h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.18)] transition-colors hover:bg-primary-hover disabled:opacity-60">
            <Plus className="h-4 w-4" />
            Tambah Webhook
          </button>
        </div>

        <section aria-labelledby="webhooks-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex flex-col gap-3 border-b border-border px-5 py-4">
            <h2 id="webhooks-heading" className="font-semibold">Daftar Endpoint Webhook</h2>
            <p className="text-xs text-muted">{pagination.total_rows} endpoint</p>
          </div>

          {loading ? (
            <div className="flex items-center justify-center gap-3 py-20" role="status">
              <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
              <span className="text-sm text-muted">Memuat webhook…</span>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <AlertTriangle className="h-10 w-10 text-error" />
              <h3 className="mt-3 text-base font-semibold">Data webhook belum dapat dimuat</h3>
              <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
              <button type="button" onClick={fetchWebhooks} className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover">Coba lagi</button>
            </div>
          ) : endpoints.length === 0 ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <Webhook className="h-10 w-10 text-muted/45" />
              <h3 className="mt-3 text-base font-semibold">Belum ada webhook</h3>
              <p className="mt-1 max-w-sm text-xs text-muted">Gunakan &quot;Tambah Webhook&quot; untuk mengirim event ke sistem eksternal.</p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[860px] border-collapse text-left">
                  <thead>
                    <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                      <th className="px-5 py-3.5">Nama</th>
                      <th className="px-5 py-3.5">URL</th>
                      <th className="px-5 py-3.5">Event</th>
                      <th className="px-5 py-3.5">Status</th>
                      <th className="px-5 py-3.5 text-right">Tindakan</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border text-sm">
                    {endpoints.map((ep) => (
                      <tr key={ep.id} className="transition-colors hover:bg-background/40">
                        <td className="px-5 py-4">
                          <p className="font-semibold">{ep.name}</p>
                          {ep.user?.full_name && <p className="text-xs text-muted">oleh {ep.user.full_name}</p>}
                        </td>
                        <td className="max-w-[260px] px-5 py-4">
                          <span className="block truncate font-mono text-xs text-primary" title={ep.url}>{ep.url}</span>
                        </td>
                        <td className="px-5 py-4">
                          <div className="flex max-w-[260px] flex-wrap gap-1">
                            {(ep.events || []).map((ev) => (
                              <span key={ev} className="rounded-full border border-border bg-background px-2 py-0.5 text-[11px] font-semibold">{EVENT_LABELS[ev] || ev}</span>
                            ))}
                          </div>
                        </td>
                        <td className="px-5 py-4">
                          <button type="button" onClick={() => handleToggle(ep)} disabled={actionLoading !== null} title="Ubah status aktif" className={`rounded-full border px-2.5 py-1 text-[11px] font-bold ${ep.is_active ? 'border-success/25 bg-success/10 text-success' : 'border-error/25 bg-error/10 text-error'}`}>
                            {ep.is_active ? 'Aktif' : 'Nonaktif'}
                          </button>
                        </td>
                        <td className="px-5 py-4 text-right">
                          <div className="flex items-center justify-end gap-1">
                            <button type="button" onClick={() => handleTest(ep)} disabled={actionLoading !== null} className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary disabled:opacity-50" title="Kirim test payload">
                              {actionLoading === `test-${ep.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                            </button>
                            <button type="button" onClick={() => openDetail(ep)} className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary" title="Lihat riwayat pengiriman">
                              <Eye className="h-4 w-4" />
                            </button>
                            <button type="button" onClick={() => openEdit(ep)} className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary" title="Ubah webhook">
                              <Pencil className="h-4 w-4" />
                            </button>
                            <button type="button" onClick={() => handleDelete(ep)} disabled={actionLoading !== null} className="rounded-lg p-2.5 text-muted transition-colors hover:bg-error/10 hover:text-error disabled:opacity-50" title="Hapus webhook">
                              {actionLoading === `delete-${ep.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Pagination page={pagination.page} limit={pagination.limit} totalRows={pagination.total_rows} totalPages={pagination.total_pages} onPageChange={setPage} itemLabel="webhook" />
            </>
          )}
        </section>
      </div>

      {/* MODAL FORM TAMBAH/EDIT WEBHOOK */}
      {isFormOpen && (
        <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => { if (!saveLoading) setIsFormOpen(false); }}></div>
          <div role="dialog" aria-modal="true" aria-labelledby="webhook-form-title" className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6">
            <div className="mb-6 flex items-start justify-between gap-4">
              <div>
                <h3 id="webhook-form-title" className="text-xl font-bold tracking-[-0.015em]">{current ? 'Ubah Webhook' : 'Tambah Webhook'}</h3>
                <p className="mt-1 text-sm text-muted">Konfigurasi endpoint penerima event eksternal.</p>
              </div>
              <button type="button" aria-label="Tutup" onClick={() => { if (!saveLoading) setIsFormOpen(false); }} disabled={saveLoading} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground disabled:opacity-50">
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label htmlFor="wh-name" className="mb-1.5 block text-xs font-semibold text-muted">Nama *</label>
                <input id="wh-name" autoFocus type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Contoh: Pusat Monitoring" className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all" required />
              </div>
              <div>
                <label htmlFor="wh-url" className="mb-1.5 block text-xs font-semibold text-muted">URL Endpoint *</label>
                <input id="wh-url" type="url" value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://sistem-eksternal.example/hook" className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all" required />
              </div>
              <div>
                <label htmlFor="wh-secret" className="mb-1.5 block text-xs font-semibold text-muted">Secret (HMAC signature)</label>
                <input id="wh-secret" type="password" value={form.secret} onChange={(e) => setForm({ ...form, secret: e.target.value })} placeholder={current ? 'Biarkan kosong agar tidak berubah' : 'Min. 8 karakter'} className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all" />
              </div>
              <div>
                <span className="mb-1.5 block text-xs font-semibold text-muted">Event yang dilanggan *</span>
                <div className="space-y-2">
                  {EVENTS.map((ev) => (
                    <label key={ev.value} className="flex cursor-pointer items-center gap-3 rounded-xl border border-border px-3 py-2.5 hover:bg-background/40">
                      <input type="checkbox" checked={form.events.includes(ev.value)} onChange={() => toggleEvent(ev.value)} className="h-4 w-4 rounded border-input-border text-primary focus:ring-primary/30" />
                      <span className="text-sm font-semibold">{ev.label}</span>
                      <code className="ml-auto font-mono text-[11px] text-muted">{ev.value}</code>
                    </label>
                  ))}
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
                <button type="button" onClick={() => { if (!saveLoading) setIsFormOpen(false); }} disabled={saveLoading} className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background disabled:opacity-50">Batal</button>
                <button type="submit" disabled={saveLoading} className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover disabled:opacity-65">
                  {saveLoading && <LoaderCircle className="h-4 w-4 animate-spin" />}
                  {saveLoading ? 'Menyimpan…' : current ? 'Simpan Perubahan' : 'Tambah Webhook'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL DETAIL + RIWAYAT PENGIRIMAN */}
      {detail && (
        <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setDetail(null)}></div>
          <div role="dialog" aria-modal="true" aria-labelledby="wh-detail-title" className="z-10 max-h-[90vh] w-full max-w-3xl overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6">
            <div className="mb-6 flex items-start justify-between gap-4">
              <div>
                <h3 id="wh-detail-title" className="text-xl font-bold tracking-[-0.015em]">{detail.endpoint.name}</h3>
                <p className="mt-1 break-all font-mono text-xs text-muted">{detail.endpoint.url}</p>
              </div>
              <button type="button" aria-label="Tutup" onClick={() => setDetail(null)} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="mb-4 flex flex-wrap gap-1.5">
              {(detail.endpoint.events || []).map((ev) => (
                <span key={ev} className="rounded-full border border-border bg-background px-2.5 py-1 text-[11px] font-semibold">{EVENT_LABELS[ev] || ev}</span>
              ))}
            </div>

            <h4 className="mb-2 flex items-center gap-2 text-sm font-semibold">
              <Zap className="h-4 w-4 text-primary" /> Riwayat Pengiriman
            </h4>
            {detail.deliveries.length === 0 ? (
              <div className="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted">Belum ada pengiriman untuk endpoint ini.</div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-border">
                <table className="w-full min-w-[700px] border-collapse text-left text-sm">
                  <thead>
                    <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                      <th className="px-4 py-3">Event</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">HTTP</th>
                      <th className="px-4 py-3">Latensi</th>
                      <th className="px-4 py-3">Waktu</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {detail.deliveries.map((d) => {
                      const badge = STATUS_BADGE[d.status] || STATUS_BADGE.pending;
                      const StatusIcon = d.status === 'success' ? CheckCircle2 : d.status === 'failed' ? XCircle : Clock;
                      return (
                        <tr key={d.id} className="hover:bg-background/40">
                          <td className="px-4 py-3 font-mono text-xs text-primary">{d.event}</td>
                          <td className="px-4 py-3">
                            <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-bold ${badge.className}`}>
                              <StatusIcon className="h-3 w-3" /> {badge.label} (percobaan {d.attempt})
                            </span>
                          </td>
                          <td className="px-4 py-3 tabular-nums">{d.response_code || '-'}</td>
                          <td className="px-4 py-3 tabular-nums">{d.latency} ms</td>
                          <td className="px-4 py-3 text-xs text-muted">{formatDate(d.created_at)}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}
