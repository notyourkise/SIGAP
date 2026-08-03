'use client';

import React, { useState, useEffect, useCallback } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { apiRequest } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import AuthenticatedImage from '@/components/AuthenticatedImage';
import { FIELD_CHECKLIST } from '@/components/FieldMaintenanceModal';
import {
  Search,
  ClipboardCheck,
  AlertTriangle,
  X,
  Eye,
  Trash2,
  LoaderCircle,
  MapPin,
  CalendarDays,
  CheckCircle2,
  User as UserIcon,
  Camera,
} from 'lucide-react';

interface FieldCheck {
  id: string;
  name: string;
  result: 'ok' | 'not_ok';
  note: string;
}

interface FieldImage {
  id: string;
  image_url: string;
}

interface FieldMaintenance {
  id: string;
  cctv_id: string;
  cctv: {
    id: string;
    kode_cctv: string;
    nama: string;
    lokasi_area: string;
    status: string;
  };
  technician_id: string;
  technician: { id: string; full_name: string; username: string };
  overall_status: 'ok' | 'attention';
  notes: string;
  completed_at: string;
  created_at: string;
  checks: FieldCheck[];
  images: FieldImage[];
}

const STATUS_META = {
  ok: {
    label: 'Normal',
    className: 'border border-emerald-500/25 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
    dot: 'bg-emerald-500',
  },
  attention: {
    label: 'Perlu Tindak Lanjut',
    className: 'border border-amber-500/25 bg-amber-500/10 text-amber-600 dark:text-amber-400',
    dot: 'bg-amber-500',
  },
} as const;

export default function FieldMaintenancePage() {
  const toast = useToast();
  const confirm = useConfirm();

  const [entries, setEntries] = useState<FieldMaintenance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({
    total_rows: 0,
    total_pages: 1,
    page: 1,
    limit: 10,
  });

  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  const [detailEntry, setDetailEntry] = useState<FieldMaintenance | null>(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const fetchEntries = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const queryParams = [`page=${page}`, 'limit=10'];
      if (search) queryParams.push(`search=${encodeURIComponent(search)}`);
      if (statusFilter) queryParams.push(`status=${encodeURIComponent(statusFilter)}`);
      if (startDate) queryParams.push(`start_date=${startDate}`);
      if (endDate) queryParams.push(`end_date=${endDate}`);

      const response = await apiRequest<any>(`/field-maintenance?${queryParams.join('&')}`);
      if (response.status === 'success' && response.data) {
        setEntries(response.data.rows || []);
        setPagination(response.data.pagination || {
          total_rows: 0,
          total_pages: 1,
          page: 1,
          limit: 10,
        });
      }
    } catch (err: any) {
      setError(err.message || 'Data pengecekan lapangan belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [page, search, statusFilter, startDate, endDate]);

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries]);

  const handleOpenDetail = async (id: string) => {
    setActionLoading(`detail-${id}`);
    try {
      const response = await apiRequest<FieldMaintenance>(`/field-maintenance/${id}`);
      if (response.data) {
        setDetailEntry(response.data);
        setIsDetailOpen(true);
      }
    } catch (err: any) {
      toast.error(err.message || 'Detail pengecekan belum dapat dimuat. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (entry: FieldMaintenance) => {
    const ok = await confirm({
      title: 'Hapus Pengecekan?',
      message: `Hapus catatan pengecekan lapangan CCTV “${entry.cctv?.nama || ''}” tanggal ${new Date(entry.created_at).toLocaleDateString('id-ID')}? Data tidak dapat dikembalikan.`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;

    setActionLoading(`delete-${entry.id}`);
    try {
      const response = await apiRequest(`/field-maintenance/${entry.id}`, { method: 'DELETE' });
      if (response.status === 'success') {
        toast.success('Catatan pengecekan lapangan berhasil dihapus.');
        fetchEntries();
      }
    } catch (err: any) {
      toast.error(err.message || 'Catatan belum dapat dihapus. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const formatDate = (iso: string) =>
    new Date(iso).toLocaleDateString('id-ID', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });

  return (
    <DashboardLayout>
      <div className="space-y-8">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Pengecekan Lapangan</h1>
            <p className="mt-1 text-sm text-muted">
              Hasil pengecekan langsung di lokasi oleh teknisi (hasil pindai QR CCTV).
            </p>
          </div>
          <span className="inline-flex w-fit items-center gap-2 rounded-xl border border-border bg-card px-4 py-2.5 text-sm font-semibold text-muted">
            <ClipboardCheck className="h-4 w-4 text-primary" />
            {pagination.total_rows} catatan
          </span>
        </header>

        <section aria-labelledby="field-maintenance-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex flex-col gap-1 border-b border-border px-5 py-4">
            <h2 id="field-maintenance-heading" className="text-lg font-bold">Daftar Hasil Pengecekan</h2>
            <p className="mt-0.5 text-xs text-muted">
              Catatan pengecekan spontan teknisi di lapangan — berbeda dari jadwal pemeliharaan rutin.
            </p>
          </div>

          <div className="grid gap-3 border-b border-border px-5 py-4 lg:grid-cols-[minmax(240px,1fr)_200px_200px_200px]">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
              <input
                type="search"
                aria-label="Cari pengecekan lapangan"
                placeholder="Cari nama CCTV, kode, atau catatan"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-xl border border-input-border bg-input py-2.5 pl-10 pr-4 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <select
              aria-label="Filter hasil pengecekan"
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(1);
              }}
              className="rounded-xl border border-input-border bg-input px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary"
            >
              <option value="">Semua hasil</option>
              <option value="ok">Normal</option>
              <option value="attention">Perlu tindak lanjut</option>
            </select>
            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-muted">Dari tanggal</label>
              <input
                type="date"
                aria-label="Filter dari tanggal"
                value={startDate}
                onChange={(e) => {
                  setStartDate(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-xl border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-muted">Sampai tanggal</label>
              <input
                type="date"
                aria-label="Filter sampai tanggal"
                value={endDate}
                onChange={(e) => {
                  setEndDate(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-xl border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary"
              />
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center gap-3 py-20" role="status">
              <div className="h-8 w-8 animate-spin rounded-full border-3 border-primary border-t-transparent"></div>
              <span className="text-sm text-muted">Memuat data pengecekan…</span>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <AlertTriangle className="h-11 w-11 text-error" />
              <h3 className="mt-3 text-base font-semibold">Data belum dapat dimuat</h3>
              <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
              <button type="button" onClick={fetchEntries} className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white">
                Coba lagi
              </button>
            </div>
          ) : entries.length === 0 ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <ClipboardCheck className="mb-3 h-10 w-10 text-muted/45" />
              <h3 className="text-base font-semibold">
                {search || statusFilter || startDate || endDate ? 'Tidak ada hasil yang cocok' : 'Belum ada catatan pengecekan'}
              </h3>
              <p className="mt-1 max-w-sm text-xs text-muted">
                {search || statusFilter || startDate || endDate
                  ? 'Ubah kata kunci atau filter untuk melihat catatan lain.'
                  : 'Catatan muncul ketika teknisi melakukan pengecekan lapangan melalui pindai QR CCTV.'}
              </p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[860px] border-collapse text-left">
                  <thead>
                    <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                      <th className="px-5 py-3.5">CCTV</th>
                      <th className="px-5 py-3.5">Teknisi</th>
                      <th className="px-5 py-3.5">Tanggal</th>
                      <th className="px-5 py-3.5">Hasil Checklist</th>
                      <th className="px-5 py-3.5">Status</th>
                      <th className="px-5 py-3.5 text-right">Tindakan</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border text-sm">
                    {entries.map((entry) => {
                      const okCount = entry.checks?.filter((c) => c.result === 'ok').length ?? 0;
                      const totalCount = entry.checks?.length ?? 0;
                      const meta = STATUS_META[entry.overall_status === 'ok' ? 'ok' : 'attention'];
                      return (
                        <tr key={entry.id} className="transition-colors hover:bg-background/35">
                          <td className="px-5 py-4">
                            <div className="min-w-0">
                              <h3 className="max-w-[260px] truncate font-semibold" title={entry.cctv?.nama}>
                                {entry.cctv?.nama || '-'}
                              </h3>
                              <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1">
                                <span className="font-mono text-xs font-semibold text-primary">{entry.cctv?.kode_cctv || '-'}</span>
                                <span className="inline-flex items-center gap-1 text-xs text-muted">
                                  <MapPin className="h-3 w-3" />
                                  {entry.cctv?.lokasi_area || '-'}
                                </span>
                              </div>
                            </div>
                          </td>
                          <td className="px-5 py-4">
                            <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-foreground">
                              <UserIcon className="h-3.5 w-3.5 text-muted" />
                              {entry.technician?.full_name || entry.technician?.username || '-'}
                            </span>
                          </td>
                          <td className="px-5 py-4">
                            <span className="inline-flex items-center gap-1.5 text-xs text-muted">
                              <CalendarDays className="h-3.5 w-3.5" />
                              {formatDate(entry.created_at)}
                            </span>
                          </td>
                          <td className="px-5 py-4">
                            <span className="inline-flex items-center gap-1.5 text-xs font-semibold">
                              <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
                              {okCount}/{totalCount} normal
                            </span>
                          </td>
                          <td className="px-5 py-4">
                            <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold ${meta.className}`}>
                              <span className={`h-1.5 w-1.5 rounded-full ${meta.dot}`} />
                              {meta.label}
                            </span>
                          </td>
                          <td className="px-5 py-4 text-right">
                            <div className="flex items-center justify-end gap-1">
                              <button
                                type="button"
                                onClick={() => handleOpenDetail(entry.id)}
                                disabled={actionLoading === `detail-${entry.id}`}
                                className="rounded-lg p-2 text-muted transition-colors hover:bg-background hover:text-foreground"
                                title="Lihat detail pengecekan"
                                aria-label={`Lihat detail pengecekan ${entry.cctv?.nama}`}
                              >
                                {actionLoading === `detail-${entry.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />}
                              </button>
                              <button
                                type="button"
                                onClick={() => handleDelete(entry)}
                                disabled={actionLoading === `delete-${entry.id}`}
                                className="rounded-lg p-2 text-muted transition-colors hover:bg-error/10 hover:text-error"
                                title="Hapus catatan"
                                aria-label={`Hapus catatan ${entry.cctv?.nama}`}
                              >
                                {actionLoading === `delete-${entry.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
              <Pagination
                page={pagination.page}
                limit={pagination.limit}
                totalRows={pagination.total_rows}
                totalPages={pagination.total_pages}
                onPageChange={(nextPage) => setPage(nextPage)}
                itemLabel="catatan"
              />
            </>
          )}
        </section>
      </div>

      {/* Detail Modal */}
      {isDetailOpen && detailEntry && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
          <div className="relative w-full max-w-2xl space-y-4 overflow-y-auto rounded-2xl border border-border bg-card p-6 shadow-2xl max-h-[92vh]">
            <div className="flex items-start justify-between gap-3 border-b border-border pb-4">
              <div className="min-w-0">
                <h3 className="text-xl font-bold">Detail Pengecekan Lapangan</h3>
                <p className="mt-0.5 truncate text-xs text-muted">
                  {detailEntry.cctv?.kode_cctv} · {detailEntry.cctv?.nama}
                </p>
              </div>
              <button
                type="button"
                aria-label="Tutup detail"
                onClick={() => setIsDetailOpen(false)}
                className="rounded-lg border border-border p-1 text-muted transition-colors hover:bg-background hover:text-foreground"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="grid gap-3 text-xs sm:grid-cols-3">
              <div className="rounded-xl border border-border bg-background p-3">
                <span className="block text-muted">Teknisi</span>
                <span className="mt-1 block font-semibold text-foreground">
                  {detailEntry.technician?.full_name || detailEntry.technician?.username || '-'}
                </span>
              </div>
              <div className="rounded-xl border border-border bg-background p-3">
                <span className="block text-muted">Waktu pengecekan</span>
                <span className="mt-1 block font-semibold text-foreground">{formatDate(detailEntry.created_at)}</span>
              </div>
              <div className="rounded-xl border border-border bg-background p-3">
                <span className="block text-muted">Hasil keseluruhan</span>
                <span className={`mt-1 inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                  STATUS_META[detailEntry.overall_status === 'ok' ? 'ok' : 'attention'].className
                }`}>
                  <span className={`h-1.5 w-1.5 rounded-full ${STATUS_META[detailEntry.overall_status === 'ok' ? 'ok' : 'attention'].dot}`} />
                  {STATUS_META[detailEntry.overall_status === 'ok' ? 'ok' : 'attention'].label}
                </span>
              </div>
            </div>

            <div>
              <h4 className="mb-2 text-sm font-bold">Hasil Checklist ({detailEntry.checks?.length ?? 0} item)</h4>
              <div className="overflow-hidden rounded-xl border border-border">
                <table className="w-full border-collapse text-left text-sm">
                  <tbody className="divide-y divide-border">
                    {detailEntry.checks?.map((check) => {
                      const checklistItem = FIELD_CHECKLIST.find((item) => item.name === check.name);
                      const isOk = check.result === 'ok';
                      return (
                        <tr key={check.id} className="align-top">
                          <td className="w-6 border-r border-border px-4 py-3">
                            <span className={`inline-flex h-5 w-5 items-center justify-center rounded-full ${isOk ? 'bg-emerald-500/15 text-emerald-500' : 'bg-red-500/15 text-red-500'}`}>
                              {isOk ? <CheckCircle2 className="h-3.5 w-3.5" /> : <AlertTriangle className="h-3.5 w-3.5" />}
                            </span>
                          </td>
                          <td className="px-4 py-3">
                            <p className="font-semibold">{check.name}</p>
                            {checklistItem && <p className="mt-0.5 text-xs text-muted">{checklistItem.desc}</p>}
                            {check.note && (
                              <p className={`mt-1.5 rounded-lg px-2.5 py-1.5 text-xs ${isOk ? 'bg-background' : 'bg-red-500/10 text-red-600 dark:text-red-400'}`}>
                                {check.note}
                              </p>
                            )}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                              isOk
                                ? 'border border-emerald-500/25 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                                : 'border border-red-500/25 bg-red-500/10 text-red-600 dark:text-red-400'
                            }`}>
                              {isOk ? 'Normal' : 'Bermasalah'}
                            </span>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>

            {detailEntry.notes && (
              <div className="rounded-xl border border-border bg-background p-4">
                <h4 className="text-sm font-bold">Catatan umum</h4>
                <p className="mt-1 whitespace-pre-wrap text-xs leading-relaxed text-foreground">{detailEntry.notes}</p>
              </div>
            )}

            <div>
              <h4 className="mb-2 text-sm font-bold">Foto Bukti ({detailEntry.images?.length ?? 0})</h4>
              {detailEntry.images && detailEntry.images.length > 0 ? (
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                  {detailEntry.images.map((img) => (
                    <div key={img.id} className="group relative aspect-square overflow-hidden rounded-xl border border-border">
                      <AuthenticatedImage
                        imageUrl={img.image_url}
                        alt={`Foto bukti pengecekan ${detailEntry.cctv?.kode_cctv}`}
                        className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
                      />
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex items-center gap-2 rounded-xl border border-dashed border-border bg-background px-4 py-6 text-xs text-muted">
                  <Camera className="h-4 w-4" />
                  Tidak ada foto bukti.
                </div>
              )}
            </div>

            <div className="flex justify-end gap-2 border-t border-border pt-4">
              <button
                type="button"
                onClick={() => setIsDetailOpen(false)}
                className="rounded-lg border border-border px-5 py-2 text-sm font-semibold transition-colors hover:bg-background"
              >
                Tutup
              </button>
              <button
                type="button"
                onClick={() => handleDelete(detailEntry)}
                disabled={actionLoading === `delete-${detailEntry.id}`}
                className="inline-flex items-center gap-2 rounded-lg bg-error px-5 py-2 text-sm font-semibold text-white transition-colors hover:bg-error/90 disabled:cursor-wait disabled:opacity-60"
              >
                {actionLoading === `delete-${detailEntry.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                Hapus Catatan
              </button>
            </div>
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}
