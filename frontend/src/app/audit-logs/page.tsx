'use client';

import React, { useState, useEffect, useCallback } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import SortableHeader from '@/components/SortableHeader';
import { useSortableRows } from '@/hooks/useSortableRows';
import { apiRequest, downloadApiFile } from '@/lib/api';
import { useAuth } from '@/context/AuthContext';
import { useWebSocket } from '@/context/WebSocketContext';
import { 
  ShieldCheck, 
  Search, 
  Eye, 
  X, 
  AlertTriangle, 
  Clock, 
  User as UserIcon, 
  Laptop, 
  Globe, 
  Activity,
  FileSpreadsheet,
  LoaderCircle,
  SlidersHorizontal,
  RotateCcw,
} from 'lucide-react';

interface AuditLog {
  id: string;
  user_id: string;
  username: string;
  role: string;
  action: string;
  module: string;
  description: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
}

interface Pagination {
  total_rows: number;
  total_pages: number;
  page: number;
  limit: number;
}

export default function AuditLogsPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState<Pagination>({
    total_rows: 0,
    total_pages: 1,
    page: 1,
    limit: 10,
  });

  // Search & Filter State
  const [search, setSearch] = useState('');
  const [moduleFilter, setModuleFilter] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  // Sort State (disembunyikan — diatur lewat modal "Sortir")
  const [sortBy, setSortBy] = useState('created_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  // Modal Sortir & Filter
  const [isSortModalOpen, setIsSortModalOpen] = useState(false);
  const [draftSortBy, setDraftSortBy] = useState('created_at');
  const [draftSortOrder, setDraftSortOrder] = useState<'asc' | 'desc'>('desc');
  const [draftModuleFilter, setDraftModuleFilter] = useState('');
  const [draftStartDate, setDraftStartDate] = useState('');
  const [draftEndDate, setDraftEndDate] = useState('');

  // Jumlah filter/sortir aktif — dipakai untuk badge pada tombol Sortir
  const activeFilterCount =
    (moduleFilter ? 1 : 0) +
    (startDate ? 1 : 0) +
    (endDate ? 1 : 0) +
    (sortBy !== 'created_at' || sortOrder !== 'desc' ? 1 : 0);

  const [exporting, setExporting] = useState(false);
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);

  // Sortir kolom (client-side) — hanya mengurutkan baris yang sudah ada di state
  const { sortedRows, sortKey, sortDirection, onSort } = useSortableRows<AuditLog>(logs, {
    getValue: (row, key) => {
      switch (key) {
        case 'waktu':
          return row.created_at;
        case 'pengguna':
          return row.username ?? '';
        case 'modul':
          return row.module ?? '';
        case 'aksi':
          return row.action ?? '';
        case 'deskripsi':
          return row.description ?? '';
        case 'ip':
          return row.ip_address ?? '';
        default:
          return '';
      }
    },
  });

  const fetchAuditLogs = useCallback(async () => {
    if (!user || !isAdmin) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError('');
    try {
      const queryParams = [`page=${page}`, 'limit=10'];
      if (search) queryParams.push(`search=${encodeURIComponent(search)}`);
      if (moduleFilter) queryParams.push(`module=${encodeURIComponent(moduleFilter)}`);
      if (startDate) queryParams.push(`start_date=${encodeURIComponent(startDate)}`);
      if (endDate) queryParams.push(`end_date=${encodeURIComponent(endDate)}`);
      if (sortBy) queryParams.push(`sort_by=${encodeURIComponent(sortBy)}`);
      if (sortOrder) queryParams.push(`order=${encodeURIComponent(sortOrder)}`);

      const response = await apiRequest<{ rows: AuditLog[]; pagination: Pagination }>(`/audit-logs?${queryParams.join('&')}`);

      if (response.status === 'success' && response.data) {
        setLogs(response.data.rows || []);
        setPagination(response.data.pagination || {
          total_rows: 0,
          total_pages: 1,
          page: 1,
          limit: 10,
        });
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Data audit log belum dapat dimuat. Coba lagi.';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [user, isAdmin, page, search, sortBy, sortOrder, moduleFilter, startDate, endDate]);

  useEffect(() => {
    void fetchAuditLogs();
  }, [fetchAuditLogs]);

  // WebSocket Live Listener
  const { subscribe } = useWebSocket();
  useEffect(() => {
    const unsub = subscribe('AUDIT_LOG_ADDED', () => fetchAuditLogs());
    return unsub;
  }, [subscribe, fetchAuditLogs]);

  const handleExport = async () => {
    setExporting(true);
    try {
      const params = new URLSearchParams();
      if (search) params.set('search', search);
      if (moduleFilter) params.set('module', moduleFilter);
      if (startDate) params.set('start_date', startDate);
      if (endDate) params.set('end_date', endDate);
      await downloadApiFile(`/audit-logs/export?${params.toString()}`, 'audit-log.csv');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Audit log belum dapat diekspor. Coba lagi.';
      setError(message);
    } finally {
      setExporting(false);
    }
  };

  const openSortModal = () => {
    setDraftSortBy(sortBy);
    setDraftSortOrder(sortOrder);
    setDraftModuleFilter(moduleFilter);
    setDraftStartDate(startDate);
    setDraftEndDate(endDate);
    setIsSortModalOpen(true);
  };

  const applySortFilter = () => {
    let changed = false;
    if (draftSortBy !== sortBy) { setSortBy(draftSortBy); changed = true; }
    if (draftSortOrder !== sortOrder) { setSortOrder(draftSortOrder); changed = true; }
    if (draftModuleFilter !== moduleFilter) { setModuleFilter(draftModuleFilter); changed = true; }
    if (draftStartDate !== startDate) { setStartDate(draftStartDate); changed = true; }
    if (draftEndDate !== endDate) { setEndDate(draftEndDate); changed = true; }
    if (changed) setPage(1);
    setIsSortModalOpen(false);
  };

  const resetSortFilter = () => {
    setSortBy('created_at');
    setSortOrder('desc');
    setModuleFilter('');
    setStartDate('');
    setEndDate('');
    setPage(1);
    setIsSortModalOpen(false);
  };

  useEffect(() => {
    if (!isSortModalOpen) return;

    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsSortModalOpen(false);
    };

    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isSortModalOpen]);

  const getModuleBadgeStyle = (mod: string) => {
    switch (mod?.toUpperCase()) {
      case 'AUTH':
        return 'border-purple-500/30 bg-purple-500/10 text-purple-600 dark:text-purple-400';
      case 'CCTV':
        return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400';
      case 'ASSET':
        return 'border-blue-500/30 bg-blue-500/10 text-blue-600 dark:text-blue-400';
      case 'MAINTENANCE':
        return 'border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400';
      case 'USER':
        return 'border-indigo-500/30 bg-indigo-500/10 text-indigo-600 dark:text-indigo-400';
      default:
        return 'border-gray-500/30 bg-gray-500/10 text-gray-600 dark:text-gray-400';
    }
  };

  const getActionBadgeStyle = (act: string) => {
    if (act?.includes('DELETE')) {
      return 'border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400';
    }
    if (act?.includes('CREATE') || act?.includes('LOGIN_SUCCESS')) {
      return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400';
    }
    if (act?.includes('UPDATE') || act?.includes('RESET')) {
      return 'border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400';
    }
    return 'border-blue-500/30 bg-blue-500/10 text-blue-600 dark:text-blue-400';
  };

  return (
    <DashboardLayout>
      <div className="space-y-8">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="flex items-center gap-2 text-primary font-semibold text-sm">
              <ShieldCheck className="h-4 w-4" />
              <span>Audit Trail Keamanan</span>
            </div>
            <h1 className="text-3xl font-bold tracking-[-0.025em] mt-1">Audit Log Sistem</h1>
            <p className="mt-1 text-sm text-muted">Rekam jejak seluruh aktivitas operasional dan administratif sistem SIGAP.</p>
          </div>
        </header>

        {!isAdmin ? (
          <div className="flex flex-col items-center justify-center rounded-2xl border border-error/20 bg-error/5 px-6 py-16 text-center">
            <AlertTriangle className="h-12 w-12 text-error" />
            <h3 className="mt-4 text-lg font-bold">Akses Ditolak</h3>
            <p className="mt-1 max-w-md text-sm text-muted">
              Halaman Audit Log HANYA dapat diakses oleh pengguna dengan kewenangan **Admin**.
            </p>
          </div>
        ) : (
          <section aria-labelledby="audit-list-heading" className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
            <div className="flex flex-col gap-1 border-b border-border px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 id="audit-list-heading" className="text-lg font-bold">Riwayat Aktivitas</h2>
                <p className="mt-0.5 text-xs text-muted">{pagination.total_rows} total entri audit log</p>
              </div>
              <button
                type="button"
                onClick={openSortModal}
                disabled={loading}
                className="relative inline-flex min-h-11 items-center gap-2 rounded-xl border border-border bg-background px-4 py-2.5 text-sm font-semibold text-foreground transition-colors hover:border-primary/40 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <SlidersHorizontal className="h-4 w-4" />
                Sortir
                {activeFilterCount > 0 && (
                  <span className="absolute -right-1.5 -top-1.5 inline-flex min-w-5 items-center justify-center rounded-full bg-primary px-1 text-[11px] font-bold leading-5 text-white">
                    {activeFilterCount}
                  </span>
                )}
              </button>
            </div>

            {/* Search Bar */}
            <div className="border-b border-border px-5 py-4">
              <div className="relative max-w-md">
                <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
                <input
                  type="search"
                  aria-label="Cari audit log"
                  placeholder="Cari deskripsi, pengguna, aksi, atau IP address..."
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setPage(1);
                  }}
                  className="w-full rounded-xl border border-input-border bg-input py-2.5 pl-10 pr-4 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
                />
              </div>
            </div>

            <div className="flex items-center justify-end gap-3 border-b border-border px-5 py-3">
              <button
                type="button"
                title="Ekspor CSV"
                aria-label="Ekspor data ke CSV"
                onClick={handleExport}
                disabled={exporting}
                className="flex min-h-10 w-10 items-center justify-center rounded-xl border border-border bg-card text-emerald-600 transition-colors hover:border-emerald-600/40 hover:bg-emerald-600/10 disabled:cursor-wait disabled:opacity-60"
              >
                {exporting ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <FileSpreadsheet className="h-4 w-4" />}
              </button>
            </div>

            {loading ? (
              <div className="flex items-center justify-center gap-3 py-20" role="status">
                <div className="h-8 w-8 animate-spin rounded-full border-3 border-primary border-t-transparent"></div>
                <span className="text-sm text-muted">Memuat riwayat audit log…</span>
              </div>
            ) : error ? (
              <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
                <AlertTriangle className="h-11 w-11 text-error" />
                <h3 className="mt-3 text-base font-semibold">Gagal memuat audit log</h3>
                <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
                <button type="button" onClick={fetchAuditLogs} className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white">
                  Coba lagi
                </button>
              </div>
            ) : logs.length === 0 ? (
              <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
                <Activity className="mb-3 h-10 w-10 text-muted/45" />
                <h3 className="text-base font-semibold">{search || moduleFilter ? 'Tidak ada log yang cocok' : 'Belum ada data audit log'}</h3>
                <p className="mt-1 max-w-sm text-xs text-muted">
                  {search || moduleFilter ? 'Ubah pencarian atau filter modul untuk melihat log lain.' : 'Seluruh aktivitas operasional sistem akan otomatis dicatat di sini.'}
                </p>
              </div>
            ) : (
              <>
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[950px] border-collapse text-left">
                    <thead>
                      <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                        <th className="px-5 py-3.5">
                          <SortableHeader label="Waktu" sortKey="waktu" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                        </th>
                        <th className="px-5 py-3.5">
                          <SortableHeader label="Pengguna" sortKey="pengguna" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                        </th>
                        <th className="px-5 py-3.5">
                          <SortableHeader label="Modul" sortKey="modul" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                        </th>
                        <th className="px-5 py-3.5">
                          <SortableHeader label="Aksi" sortKey="aksi" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                        </th>
                        <th className="px-5 py-3.5">
                          <SortableHeader label="Deskripsi Aktivitas" sortKey="deskripsi" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                        </th>
                        <th className="px-5 py-3.5">
                          <SortableHeader label="Client IP" sortKey="ip" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                        </th>
                        <th className="px-5 py-3.5 text-right">Detail</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border text-sm">
                      {sortedRows.map((log) => (
                        <tr key={log.id} className="transition-colors hover:bg-background/35">
                          <td className="whitespace-nowrap px-5 py-4 text-xs font-mono text-muted">
                            {new Date(log.created_at).toLocaleString('id-ID', {
                              year: 'numeric',
                              month: 'short',
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
                              second: '2-digit'
                            })}
                          </td>
                          <td className="px-5 py-4">
                            <div className="flex items-center gap-2">
                              <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold text-xs">
                                {log.username ? log.username.charAt(0).toUpperCase() : '?'}
                              </div>
                              <div>
                                <p className="font-semibold text-xs">{log.username || 'System / Guest'}</p>
                                <span className="text-[10px] text-muted capitalize">{log.role || '—'}</span>
                              </div>
                            </div>
                          </td>
                          <td className="px-5 py-4">
                            <span className={`inline-flex items-center rounded-md border px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wider ${getModuleBadgeStyle(log.module)}`}>
                              {log.module}
                            </span>
                          </td>
                          <td className="px-5 py-4">
                            <span className={`inline-flex items-center rounded-md border px-2.5 py-0.5 text-[11px] font-mono font-semibold ${getActionBadgeStyle(log.action)}`}>
                              {log.action}
                            </span>
                          </td>
                          <td className="max-w-xs truncate px-5 py-4 text-xs text-foreground" title={log.description}>
                            {log.description}
                          </td>
                          <td className="whitespace-nowrap px-5 py-4 font-mono text-xs text-muted">
                            {log.ip_address || '—'}
                          </td>
                          <td className="px-5 py-4 text-right">
                            <button
                              type="button"
                              onClick={() => {
                                setSelectedLog(log);
                                setIsDetailModalOpen(true);
                              }}
                              className="rounded-lg p-2 text-muted transition-colors hover:bg-background hover:text-foreground"
                              title="Lihat detail audit log"
                              aria-label={`Lihat detail audit log ${log.action}`}
                            >
                              <Eye className="h-4 w-4" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                <Pagination
                  page={pagination.page}
                  limit={pagination.limit}
                  totalRows={pagination.total_rows}
                  totalPages={pagination.total_pages}
                  onPageChange={(nextPage) => setPage(nextPage)}
                  itemLabel="Audit Log"
                />
              </>
            )}
          </section>
        )}

        {/* Modal Sortir & Filter */}
        {isSortModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setIsSortModalOpen(false)} />

            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="audit-sort-title"
              className="z-10 max-h-[90vh] w-full max-w-md overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
            >
              <div className="mb-5 flex items-start justify-between gap-4">
                <div>
                  <h3 id="audit-sort-title" className="text-xl font-bold tracking-[-0.015em]">Sortir &amp; Filter</h3>
                  <p className="mt-1 text-sm text-muted">Urutkan dan saring riwayat aktivitas.</p>
                </div>
                <button
                  type="button"
                  aria-label="Tutup sortir dan filter"
                  onClick={() => setIsSortModalOpen(false)}
                  className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <label htmlFor="audit-sort-by" className="mb-1.5 block text-xs font-semibold text-muted">Urutkan berdasarkan</label>
                  <select
                    id="audit-sort-by"
                    value={draftSortBy}
                    onChange={(e) => setDraftSortBy(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  >
                    <option value="created_at">Waktu kejadian</option>
                    <option value="action">Aksi</option>
                    <option value="module">Modul</option>
                    <option value="username">Pengguna</option>
                    <option value="user_role">Peran</option>
                  </select>
                </div>

                <div>
                  <label htmlFor="audit-sort-order" className="mb-1.5 block text-xs font-semibold text-muted">Arah pengurutan</label>
                  <select
                    id="audit-sort-order"
                    value={draftSortOrder}
                    onChange={(e) => setDraftSortOrder(e.target.value as 'asc' | 'desc')}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  >
                    <option value="desc">Terbaru ke terlama</option>
                    <option value="asc">Terlama ke terbaru</option>
                  </select>
                </div>

                <div className="space-y-4 border-t border-border pt-4">
                  <p className="text-xs font-semibold text-muted">Filter</p>
                  <div>
                    <label htmlFor="audit-sort-module-filter" className="mb-1.5 block text-xs font-semibold text-muted">Modul</label>
                    <select
                      id="audit-sort-module-filter"
                      value={draftModuleFilter}
                      onChange={(e) => setDraftModuleFilter(e.target.value)}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    >
                      <option value="">Semua modul</option>
                      <option value="AUTH">Auth</option>
                      <option value="CCTV">CCTV</option>
                      <option value="MAINTENANCE">Pemeliharaan</option>
                      <option value="USER">Manajemen User</option>
                    </select>
                  </div>

                  <div>
                    <label htmlFor="audit-sort-date-filter" className="mb-1.5 block text-xs font-semibold text-muted">Rentang tanggal kejadian</label>
                    <div className="flex items-center gap-2">
                      <input
                        id="audit-sort-date-filter"
                        type="date"
                        aria-label="Rentang tanggal mulai"
                        value={draftStartDate}
                        onChange={(e) => setDraftStartDate(e.target.value)}
                        className="min-h-11 w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      />
                      <span className="text-xs text-muted">s.d.</span>
                      <input
                        type="date"
                        aria-label="Rentang tanggal akhir"
                        value={draftEndDate}
                        onChange={(e) => setDraftEndDate(e.target.value)}
                        className="min-h-11 w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      />
                    </div>
                  </div>
                </div>
              </div>

              <div className="mt-6 flex flex-col-reverse gap-2 border-t border-border pt-4 sm:flex-row sm:justify-between">
                <button
                  type="button"
                  onClick={resetSortFilter}
                  className="inline-flex items-center justify-center gap-2 rounded-xl border border-border px-4 py-2.5 text-sm font-semibold text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  <RotateCcw className="h-4 w-4" />
                  Reset
                </button>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setIsSortModalOpen(false)}
                    className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                  >
                    Batal
                  </button>
                  <button
                    type="button"
                    onClick={applySortFilter}
                    className="rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
                  >
                    Terapkan
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Modal Detail Audit Log */}
        {isDetailModalOpen && selectedLog && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="relative w-full max-w-lg bg-card border border-border rounded-3xl p-6 shadow-2xl space-y-4 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between border-b border-border pb-4">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                    <ShieldCheck className="h-5 w-5" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold">Detail Audit Log</h3>
                    <p className="font-mono text-xs text-primary">{selectedLog.id}</p>
                  </div>
                </div>
                <button
                  type="button"
                  aria-label="Tutup detail log"
                  onClick={() => setIsDetailModalOpen(false)}
                  className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>

              <div className="space-y-4 text-sm">
                <div className="space-y-1">
                  <span className="text-xs font-semibold text-muted">Deskripsi Aktivitas</span>
                  <p className="rounded-xl border border-border bg-input p-3 text-xs leading-relaxed font-semibold">
                    {selectedLog.description}
                  </p>
                </div>

                <dl className="grid overflow-hidden rounded-2xl border border-border text-xs sm:grid-cols-2">
                  <div className="border-b border-border p-3.5 sm:border-r">
                    <dt className="flex items-center gap-1.5 text-muted">
                      <UserIcon className="h-3.5 w-3.5" /> Pengguna
                    </dt>
                    <dd className="mt-1 font-bold">{selectedLog.username || 'System / Guest'}</dd>
                    <dd className="text-[10px] text-muted capitalize">{selectedLog.role || '-'}</dd>
                  </div>

                  <div className="border-b border-border p-3.5">
                    <dt className="flex items-center gap-1.5 text-muted">
                      <Activity className="h-3.5 w-3.5" /> Modul & Aksi
                    </dt>
                    <dd className="mt-1 font-mono font-bold text-primary">{selectedLog.action}</dd>
                    <dd className="text-[10px] text-muted">Modul: {selectedLog.module}</dd>
                  </div>

                  <div className="border-b border-border p-3.5 sm:border-r">
                    <dt className="flex items-center gap-1.5 text-muted">
                      <Globe className="h-3.5 w-3.5" /> Client IP Address
                    </dt>
                    <dd className="mt-1 font-mono font-semibold">{selectedLog.ip_address || '—'}</dd>
                  </div>

                  <div className="border-b border-border p-3.5">
                    <dt className="flex items-center gap-1.5 text-muted">
                      <Clock className="h-3.5 w-3.5" /> Waktu Kejadian
                    </dt>
                    <dd className="mt-1 font-mono text-[11px] font-semibold">
                      {new Date(selectedLog.created_at).toLocaleString('id-ID', {
                        year: 'numeric',
                        month: 'long',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit'
                      })}
                    </dd>
                  </div>

                  <div className="p-3.5 sm:col-span-2">
                    <dt className="flex items-center gap-1.5 text-muted mb-1">
                      <Laptop className="h-3.5 w-3.5" /> Browser User Agent
                    </dt>
                    <dd className="font-mono text-[10px] leading-relaxed text-muted break-all rounded-lg bg-input p-2 border border-input-border">
                      {selectedLog.user_agent || 'Tidak tersedia'}
                    </dd>
                  </div>
                </dl>
              </div>

              <div className="flex justify-end pt-2">
                <button
                  type="button"
                  onClick={() => setIsDetailModalOpen(false)}
                  className="rounded-xl bg-primary px-5 py-2.5 text-xs font-semibold text-white shadow-md hover:bg-primary-hover transition-colors"
                >
                  Tutup
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}
