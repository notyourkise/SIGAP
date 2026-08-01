'use client';

import { useCallback, useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import DashboardAnalytics, { type DashboardSummary } from '@/components/DashboardAnalytics';
import { useAuth } from '@/context/AuthContext';
import { useWebSocket } from '@/context/WebSocketContext';
import { apiRequest } from '@/lib/api';

const CCTVMap = dynamic(() => import('@/components/CCTVMap'), {
  ssr: false,
  loading: () => (
    <div className="flex h-[460px] w-full items-center justify-center rounded-3xl border border-border bg-card">
      <div className="flex items-center gap-3">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        <span className="text-sm font-semibold text-muted">Memuat peta GIS IKN…</span>
      </div>
    </div>
  ),
});

interface CCTV {
  id: string;
  status: string;
  nama: string;
  kode_cctv: string;
  lokasi_area: string;
}

type CCTVFilter = 'all' | 'active' | 'inactive';
const CCTV_PAGE_SIZE = 6;

interface MaintenanceLog {
  id: string;
  status: 'scheduled' | 'ongoing' | 'completed' | 'failed';
  action_taken: string;
  created_at: string;
  cctv?: {
    nama?: string;
    lokasi_area?: string;
  };
}

interface PaginationPayload<T> {
  rows: T[];
  pagination: {
    total_rows: number;
  };
}

interface DashboardTask {
  id: string;
  cctv_name: string;
  cctv_location: string;
  status: 'scheduled' | 'ongoing';
  action_taken: string;
  scheduled_date: string;
  days_overdue: number;
}

interface DashboardTasksResponse {
  tasks: DashboardTask[];
  pending_count: number;
  overdue_count: number;
}

const normalizeStatus = (status: string) => status.trim().toUpperCase();
const isActiveCCTV = (status: string) => ['ACTIVE', 'AKTIF'].includes(normalizeStatus(status));
const isInactiveCCTV = (status: string) => ['INACTIVE', 'NONAKTIF', 'BROKEN', 'RUSAK'].includes(normalizeStatus(status));
const cctvStatusLabel = (status: string) => {
  if (isActiveCCTV(status)) return 'Aktif';
  if (isInactiveCCTV(status)) return 'Nonaktif';
  return status || 'Belum ditentukan';
};
const statusLabel = (status: MaintenanceLog['status']) => ({
  scheduled: 'Terjadwal',
  ongoing: 'Dalam proses',
  completed: 'Selesai',
  failed: 'Gagal',
}[status]);

export default function DashboardPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const [cctvs, setCctvs] = useState<CCTV[]>([]);
  const [logs, setLogs] = useState<MaintenanceLog[]>([]);
  const [totalCctv, setTotalCctv] = useState(0);
  const [totalLogs, setTotalLogs] = useState(0);
  const [selectedCctvFilter, setSelectedCctvFilter] = useState<CCTVFilter | null>(null);
  const [selectedCctvPage, setSelectedCctvPage] = useState(1);
  const [selectedStatus, setSelectedStatus] = useState<MaintenanceLog['status'] | null>(null);
  const [health, setHealth] = useState<'Memeriksa' | 'Terhubung' | 'Tidak terhubung'>('Memeriksa');
  const [analytics, setAnalytics] = useState<DashboardSummary | null>(null);
  const [tasks, setTasks] = useState<DashboardTasksResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadDashboard = useCallback(async () => {
    setLoading(true);
    setError('');
    setHealth('Memeriksa');
    try {
      const [cctvResult, maintenanceResult, healthResult, summaryResult, tasksResult] = await Promise.allSettled([
        apiRequest<PaginationPayload<CCTV>>('/cctv?page=1&limit=1000&sort_by=nama&order=asc'),
        apiRequest<PaginationPayload<MaintenanceLog>>('/maintenance-logs?page=1&limit=100'),
        apiRequest<Record<string, unknown>>('/health'),
        apiRequest<DashboardSummary>('/dashboard/summary'),
        apiRequest<DashboardTasksResponse>('/dashboard/tasks'),
      ]);

      if (cctvResult.status === 'fulfilled') {
        const cctvData = cctvResult.value.data;
        const cctvRows = cctvData?.rows || [];
        setCctvs(cctvRows);
        setTotalCctv(cctvData?.pagination?.total_rows ?? cctvRows.length);
      } else {
        setError(cctvResult.reason instanceof Error ? cctvResult.reason.message : 'Data CCTV belum dapat dimuat. Muat ulang dashboard.');
      }

      if (maintenanceResult.status === 'fulfilled') {
        const maintenanceData = maintenanceResult.value.data;
        const maintenanceRows = maintenanceData?.rows || [];
        setLogs(maintenanceRows);
        setTotalLogs(maintenanceData?.pagination?.total_rows ?? maintenanceRows.length);
      } else {
        setError((current) => current || (maintenanceResult.reason instanceof Error ? maintenanceResult.reason.message : 'Data pemeliharaan belum dapat dimuat. Muat ulang dashboard.'));
      }

      if (summaryResult.status === 'fulfilled') {
        setAnalytics(summaryResult.value.data ?? null);
      }

      if (tasksResult.status === 'fulfilled') {
        setTasks(tasksResult.value.data ?? null);
      }

      setHealth(healthResult.status === 'fulfilled' ? 'Terhubung' : 'Tidak terhubung');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(loadDashboard, 0);
    return () => clearTimeout(timer);
  }, [loadDashboard]);

  const { subscribe } = useWebSocket();

  useEffect(() => {
    const unsubCCTV = subscribe('CCTV_UPDATED', () => loadDashboard());
    const unsubMaintCreated = subscribe('MAINTENANCE_CREATED', () => loadDashboard());
    const unsubMaintUpdated = subscribe('MAINTENANCE_UPDATED', () => loadDashboard());

    return () => {
      unsubCCTV();
      unsubMaintCreated();
      unsubMaintUpdated();
    };
  }, [subscribe, loadDashboard]);

  const active = cctvs.filter((item) => isActiveCCTV(item.status)).length;
  const inactive = cctvs.filter((item) => isInactiveCCTV(item.status)).length;
  const scheduled = logs.filter((item) => item.status === 'scheduled').length;
  const ongoing = logs.filter((item) => item.status === 'ongoing').length;
  const completed = logs.filter((item) => item.status === 'completed').length;
  const failed = logs.filter((item) => item.status === 'failed').length;
  const selectedLogs = selectedStatus ? logs.filter((item) => item.status === selectedStatus) : [];

  const summary: Array<{ key: CCTVFilter; label: string; value: number; tone: string }> = [
    { key: 'all', label: 'Total CCTV', value: totalCctv, tone: 'bg-blue-500' },
    { key: 'active', label: 'Aktif', value: active, tone: 'bg-success' },
    { key: 'inactive', label: 'Nonaktif', value: inactive, tone: 'bg-error' },
  ];
  const selectedCctvs = selectedCctvFilter === 'active'
    ? cctvs.filter((item) => isActiveCCTV(item.status))
    : selectedCctvFilter === 'inactive'
      ? cctvs.filter((item) => isInactiveCCTV(item.status))
      : selectedCctvFilter === 'all'
        ? cctvs
        : [];
  const selectedSummary = summary.find((item) => item.key === selectedCctvFilter);
  const selectedCctvTotalPages = Math.max(1, Math.ceil(selectedCctvs.length / CCTV_PAGE_SIZE));
  const safeSelectedCctvPage = Math.min(selectedCctvPage, selectedCctvTotalPages);
  const selectedCctvStart = (safeSelectedCctvPage - 1) * CCTV_PAGE_SIZE;
  const visibleCctvs = selectedCctvs.slice(selectedCctvStart, selectedCctvStart + CCTV_PAGE_SIZE);

  return (
    <DashboardLayout>
      <div className="space-y-8">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Dashboard CCTV</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              {isAdmin ? 'Pantau kondisi aset dan pekerjaan pemeliharaan.' : 'Pantau kondisi CCTV dan prioritas pekerjaan.'}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 text-xs font-semibold">
              <span className={`h-2 w-2 rounded-full ${health === 'Terhubung' ? 'bg-success' : health === 'Memeriksa' ? 'animate-pulse bg-warning' : 'bg-error'}`} />
              <span>{health === 'Terhubung' ? 'Layanan normal' : health}</span>
            </div>
            <button
              type="button"
              onClick={loadDashboard}
              disabled={loading}
              className="rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold transition-colors hover:border-primary disabled:cursor-wait disabled:opacity-60"
            >
              {loading ? 'Memuat…' : 'Muat ulang'}
            </button>
          </div>
        </header>

        {error && <div role="alert" className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">{error}</div>}
        {!loading && health === 'Tidak terhubung' && (
          <div role="alert" className="rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm">
            Layanan SIGAP tidak dapat dijangkau. Periksa koneksi atau hubungi pengelola sistem.
          </div>
        )}

        {loading ? (
          <DashboardSkeleton />
        ) : (
          <>
            <section aria-label="Ringkasan CCTV" className="overflow-hidden rounded-2xl border border-border bg-card">
              <div className="grid divide-y divide-border sm:grid-cols-3 sm:divide-x sm:divide-y-0">
                {summary.map(({ key, label, value, tone }) => {
                  const isSelected = selectedCctvFilter === key;
                  return (
                  <button
                    key={key}
                    type="button"
                    aria-pressed={isSelected}
                    aria-expanded={isSelected}
                    aria-controls="cctv-summary-details"
                    onClick={() => {
                      setSelectedCctvFilter((current) => current === key ? null : key);
                      setSelectedCctvPage(1);
                    }}
                    className={`flex min-h-20 items-center justify-between gap-6 px-5 py-5 text-left transition-colors hover:bg-background focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary ${
                      isSelected ? 'bg-primary/10 text-foreground' : 'bg-card'
                    }`}
                  >
                    <div className="flex items-center gap-2.5">
                      <span className={`h-2.5 w-2.5 rounded-full ${tone}`} />
                      <span className={`text-sm font-medium ${isSelected ? 'text-foreground' : 'text-muted'}`}>{label}</span>
                    </div>
                    <strong className="text-2xl font-bold tabular-nums">{value}</strong>
                  </button>
                  );
                })}
              </div>
            </section>

            {/* Peta GIS Interaktif IKN */}
            <section aria-label="Peta GIS Sebaran CCTV IKN">
              <CCTVMap cctvs={cctvs} />
            </section>

            {/* Analitik: tren pemeliharaan, distribusi status, peringatan overdue */}
            <DashboardAnalytics summary={analytics} loading={loading} />

            {/* Tugas & Notifikasi: pekerjaan terjadwal/berjalan untuk teknisi */}
            <section aria-labelledby="tasks-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
              <div className="flex flex-col gap-3 border-b border-border px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h2 id="tasks-heading" className="text-base font-bold">
                    {isAdmin ? 'Tugas Berjalan Semua Teknisi' : 'Tugas Saya'}
                  </h2>
                  <p className="mt-0.5 text-xs text-muted">
                    {isAdmin
                      ? 'Pekerjaan terjadwal dan dalam proses di seluruh teknisi.'
                      : 'Pekerjaan yang dijadwalkan untuk Anda, beserta prioritas yang melewati jadwal.'}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2 text-xs font-semibold">
                  <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-background px-3 py-1.5">
                    <span className="h-2 w-2 rounded-full bg-blue-500" />
                    {tasks?.pending_count ?? 0} tugas aktif
                  </span>
                  <span className="inline-flex items-center gap-1.5 rounded-full border border-error/25 bg-error/10 px-3 py-1.5 text-error">
                    <span className="h-2 w-2 rounded-full bg-error" />
                    {tasks?.overdue_count ?? 0} terlambat
                  </span>
                </div>
              </div>

              {!tasks || tasks.tasks.length === 0 ? (
                <p className="px-5 py-8 text-sm text-muted">
                  Tidak ada tugas aktif saat ini.{isAdmin ? '' : ' Pekerjaan baru yang dijadwalkan untuk Anda akan tampil di sini.'}
                </p>
              ) : (
                <div className="divide-y divide-border">
                  {tasks.tasks.map((task) => {
                    const isOverdue = task.days_overdue > 0;
                    return (
                      <div key={task.id} className="grid gap-2 px-5 py-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-center sm:gap-4">
                        <div className="min-w-0">
                          <p className="truncate text-sm font-semibold" title={task.cctv_name}>
                            {task.cctv_name || 'CCTV tidak tersedia'}
                          </p>
                          <p className="mt-0.5 truncate text-xs text-muted" title={task.cctv_location || undefined}>
                            {task.cctv_location || 'Lokasi belum dicatat'}
                          </p>
                        </div>
                        <div className="min-w-0">
                          <p className="truncate text-xs text-muted" title={task.action_taken}>
                            {task.action_taken || '—'}
                          </p>
                          <p className="mt-0.5 font-mono text-[11px] text-muted">
                            {task.scheduled_date ? `Jadwal: ${task.scheduled_date}` : 'Belum ada jadwal'}
                          </p>
                        </div>
                        <div className="flex flex-wrap items-center gap-2">
                          {isOverdue && (
                            <span className="inline-flex items-center gap-1 rounded-full border border-error/25 bg-error/10 px-2.5 py-1 text-[11px] font-bold text-error">
                              Terlambat {task.days_overdue} hari
                            </span>
                          )}
                          <span className={`inline-flex items-center rounded-full border px-2.5 py-1 text-[11px] font-bold ${
                            task.status === 'ongoing'
                              ? 'border-warning/25 bg-warning/10 text-warning'
                              : 'border-blue-500/25 bg-blue-500/10 text-blue-500'
                          }`}>
                            {statusLabel(task.status)}
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </section>

            {selectedCctvFilter && (
              <section
                id="cctv-summary-details"
                aria-labelledby="cctv-summary-details-heading"
                className="overflow-hidden rounded-2xl border border-border bg-card"
              >
                <div className="flex items-center justify-between gap-4 border-b border-border px-5 py-4">
                  <div>
                    <h2 id="cctv-summary-details-heading" className="text-base font-bold">
                      {selectedCctvFilter === 'all' ? 'Rincian seluruh CCTV' : `Rincian CCTV ${selectedSummary?.label.toLowerCase()}`}
                    </h2>
                    <p className="mt-0.5 text-xs text-muted">Informasi perangkat yang perlu dipantau.</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedCctvFilter(null);
                      setSelectedCctvPage(1);
                    }}
                    className="rounded-lg px-2.5 py-2 text-xs font-semibold text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                  >
                    Tutup
                  </button>
                </div>

                {visibleCctvs.length === 0 ? (
                  <p className="px-5 py-6 text-sm text-muted">
                    Tidak ada CCTV dalam kategori ini.
                  </p>
                ) : (
                  <div className="divide-y divide-border">
                    {visibleCctvs.map((cctv) => {
                      const isActive = isActiveCCTV(cctv.status);
                      const isInactive = isInactiveCCTV(cctv.status);
                      return (
                        <div
                          key={cctv.id}
                          className="grid gap-2 px-5 py-3.5 sm:grid-cols-[minmax(0,1fr)_minmax(180px,0.7fr)_auto] sm:items-center sm:gap-4"
                        >
                          <div className="min-w-0">
                            <p className="truncate text-sm font-semibold" title={cctv.nama}>{cctv.nama}</p>
                            <p className="mt-0.5 font-mono text-xs font-semibold text-primary">{cctv.kode_cctv}</p>
                          </div>
                          <p className="truncate text-xs text-muted" title={cctv.lokasi_area || undefined}>
                            {cctv.lokasi_area || 'Lokasi belum dicatat'}
                          </p>
                          <span className={`inline-flex w-fit items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold ${
                            isActive
                              ? 'border-success/25 bg-success/10 text-success'
                              : isInactive
                                ? 'border-error/25 bg-error/10 text-error'
                                : 'border-warning/25 bg-warning/10 text-warning'
                          }`}>
                            <span className={`h-1.5 w-1.5 rounded-full ${isActive ? 'bg-success' : isInactive ? 'bg-error' : 'bg-warning'}`} />
                            {cctvStatusLabel(cctv.status)}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                )}

                <Pagination
                  page={safeSelectedCctvPage}
                  limit={CCTV_PAGE_SIZE}
                  totalRows={selectedCctvs.length}
                  totalPages={selectedCctvTotalPages}
                  onPageChange={setSelectedCctvPage}
                  itemLabel="CCTV"
                />
              </section>
            )}

            <section aria-labelledby="maintenance-heading">
              <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
                <div>
                  <h2 id="maintenance-heading" className="text-xl font-bold">
                    {isAdmin ? 'Pemeliharaan' : 'Prioritas Pemeliharaan'}
                  </h2>
                  <p className="mt-1 text-sm text-muted">Pilih status untuk melihat CCTV terkait.</p>
                </div>
                <p className="text-sm font-semibold text-muted">{totalLogs} catatan</p>
              </div>

              <div className="mt-4 grid overflow-hidden rounded-2xl border border-border bg-card sm:grid-cols-2 lg:grid-cols-4">
                <StatusCard label="Terjadwal" value={scheduled} tone="blue" active={selectedStatus === 'scheduled'} onClick={() => setSelectedStatus((current) => current === 'scheduled' ? null : 'scheduled')} />
                <StatusCard label="Dalam proses" value={ongoing} tone="warning" active={selectedStatus === 'ongoing'} onClick={() => setSelectedStatus((current) => current === 'ongoing' ? null : 'ongoing')} />
                <StatusCard label="Selesai" value={completed} tone="success" active={selectedStatus === 'completed'} onClick={() => setSelectedStatus((current) => current === 'completed' ? null : 'completed')} />
                <StatusCard label="Gagal" value={failed} tone="error" active={selectedStatus === 'failed'} onClick={() => setSelectedStatus((current) => current === 'failed' ? null : 'failed')} />
              </div>

              {selectedStatus && (
                <div className="mt-4 overflow-hidden rounded-2xl border border-border bg-card">
                  <div className="flex items-center justify-between border-b border-border px-5 py-4">
                    <h3 className="text-sm font-semibold">CCTV — {statusLabel(selectedStatus)}</h3>
                    <button type="button" onClick={() => setSelectedStatus(null)} className="text-xs font-semibold text-muted hover:text-foreground">Tutup</button>
                  </div>
                  {selectedLogs.length === 0 ? (
                    <p className="px-5 py-6 text-sm text-muted">
                      Belum ada pekerjaan berstatus {statusLabel(selectedStatus).toLowerCase()}.
                    </p>
                  ) : (
                    <div className="divide-y divide-border">
                      {selectedLogs.map((log) => (
                        <div key={log.id} className="grid gap-2 px-5 py-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                          <div>
                            <p className="text-sm font-semibold">{log.cctv?.nama || 'CCTV tidak tersedia'}</p>
                            <p className="mt-0.5 text-xs text-muted">{log.cctv?.lokasi_area || 'Lokasi belum dicatat'}</p>
                          </div>
                          <p className="text-sm text-muted sm:text-right">{log.action_taken}</p>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </section>
          </>
        )}
      </div>
    </DashboardLayout>
  );
}

function StatusCard({ label, value, tone, active, onClick }: {
  label: string;
  value: number;
  tone: 'success' | 'warning' | 'error' | 'blue';
  active: boolean;
  onClick: () => void;
}) {
  const tones = {
    success: 'bg-success',
    warning: 'bg-warning',
    error: 'bg-error',
    blue: 'bg-blue-500',
  };
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`flex w-full items-center justify-between gap-4 border-b border-border px-5 py-4 text-left transition-colors last:border-b-0 hover:bg-background sm:[&:nth-child(odd)]:border-r lg:border-b-0 lg:border-r lg:last:border-r-0 ${
        active ? 'bg-primary/10 text-foreground' : 'bg-card'
      }`}
    >
      <div className="flex min-w-0 items-center gap-2.5">
        <span className={`h-2.5 w-2.5 shrink-0 rounded-full ${tones[tone]}`} />
        <span className="text-sm font-semibold">{label}</span>
      </div>
      <strong className="text-xl tabular-nums">{value}</strong>
    </button>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-8" role="status" aria-label="Memuat ringkasan dashboard">
      <div className="h-24 animate-pulse rounded-2xl border border-border bg-card" />
      <div className="space-y-4">
        <div className="h-12 w-52 animate-pulse rounded-xl bg-card" />
        <div className="h-20 animate-pulse rounded-2xl border border-border bg-card" />
      </div>
    </div>
  );
}
