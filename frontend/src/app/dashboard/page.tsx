'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import DashboardAnalytics, { type DashboardSummary } from '@/components/DashboardAnalytics';
import { useAuth } from '@/context/AuthContext';
import { useWebSocket } from '@/context/WebSocketContext';
import { apiRequest } from '@/lib/api';
import { AlertTriangle, CalendarClock, CheckCheck, MapPin, Package, ScanLine } from 'lucide-react';
import { getEffectiveStreamState, getLiveBadge, type StreamState } from '@/lib/live-status';
import Link from 'next/link';
import FieldMaintenanceModal from '@/components/FieldMaintenanceModal';
import NotificationBell from '@/components/NotificationBell';
import ProfileMenu from '@/components/ProfileMenu';

interface CCTV {
  id: string;
  status: string;
  nama: string;
  kode_cctv: string;
  lokasi_area: string;
  stream_live?: boolean | null;
  stream_state?: StreamState | null;
  stream_last_active_at?: string | null;
  network_type?: 'metro-e' | 'gsm' | null;
  stream_verified_state?: StreamState | null;
  stream_verified_at?: string | null;
  health_score?: number;
}

// Payload aman dari event CCTV_LIVE_STATUS_UPDATED (probe/verifikasi).
// Event tidak membawa lokasi/status aset, jadi field tersebut dipertahankan
// dari state yang sudah ada.
interface CCTVLiveStatusPayload {
  cctv_id?: string;
  nama?: string;
  kode_cctv?: string;
  stream_live?: boolean | null;
  stream_state?: StreamState | null;
  stream_last_active_at?: string | null;
  network_type?: 'metro-e' | 'gsm' | null;
  stream_verified_state?: StreamState | null;
  stream_verified_at?: string | null;
}

type CCTVFilter = 'all' | 'live' | 'idle' | 'offline' | 'maintenance';
const CCTV_PAGE_SIZE = 6;
const REALTIME_REFRESH_DEBOUNCE_MS = 800;

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
  cctv_id: string;
  cctv_name: string;
  cctv_kode: string;
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

// Suku cadang stok rendah untuk widget dashboard (admin only).
interface LowStockSpare {
  id: string;
  nama: string;
  sku: string;
  kategori: string;
  stok: number;
  minimum_order: number;
  satuan: string;
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
  failed: 'Perlu Tindak Lanjut',
}[status]);

const formatTaskSchedule = (value: string) => {
  if (!value) return 'Jadwal belum ditentukan';
  const parsed = new Date(value.replace(' ', 'T'));
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(parsed);
};

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
  const [completeTask, setCompleteTask] = useState<DashboardTask | null>(null);
  const [lowStock, setLowStock] = useState<LowStockSpare[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const cctvsRef = useRef<CCTV[]>([]);
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const requestSeqRef = useRef(0);

  const loadDashboard = useCallback(async (options: { silent?: boolean } = {}) => {
    const { silent = false } = options;
    const requestId = ++requestSeqRef.current;

    if (!silent) {
      setLoading(true);
      setError('');
      setHealth('Memeriksa');
    }

    try {
      const [cctvResult, maintenanceResult, healthResult, summaryResult, tasksResult, lowStockResult] = await Promise.allSettled([
        apiRequest<PaginationPayload<CCTV>>('/cctv?page=1&limit=1000&sort_by=nama&order=asc'),
        apiRequest<PaginationPayload<MaintenanceLog>>('/maintenance-logs?page=1&limit=100'),
        apiRequest<Record<string, unknown>>('/health'),
        apiRequest<DashboardSummary>('/dashboard/summary'),
        apiRequest<DashboardTasksResponse>('/dashboard/tasks'),
        // Widget suku cadang stok rendah — hanya untuk admin.
        isAdmin ? apiRequest<LowStockSpare[]>('/spare-parts/low-stock') : Promise.resolve(null),
      ]);

      // Abaikan respons lama agar tidak menimpa data yang lebih baru.
      if (requestId !== requestSeqRef.current) return;

      let nextError = '';

      if (cctvResult.status === 'fulfilled') {
        const cctvData = cctvResult.value.data;
        const cctvRows = cctvData?.rows || [];
        cctvsRef.current = cctvRows;
        setCctvs(cctvRows);
        setTotalCctv(cctvData?.pagination?.total_rows ?? cctvRows.length);
      } else {
        nextError = cctvResult.reason instanceof Error ? cctvResult.reason.message : 'Data CCTV belum dapat dimuat. Muat ulang dashboard.';
      }

      if (maintenanceResult.status === 'fulfilled') {
        const maintenanceData = maintenanceResult.value.data;
        const maintenanceRows = maintenanceData?.rows || [];
        setLogs(maintenanceRows);
        setTotalLogs(maintenanceData?.pagination?.total_rows ?? maintenanceRows.length);
      } else {
        nextError = nextError || (maintenanceResult.reason instanceof Error ? maintenanceResult.reason.message : 'Data pemeliharaan belum dapat dimuat. Muat ulang dashboard.');
      }

      if (summaryResult.status === 'fulfilled') {
        setAnalytics(summaryResult.value.data ?? null);
      }

      if (tasksResult.status === 'fulfilled') {
        setTasks(tasksResult.value.data ?? null);
      }

      if (isAdmin && lowStockResult && lowStockResult.status === 'fulfilled') {
        const data = lowStockResult.value?.data;
        setLowStock(Array.isArray(data) ? data : []);
      }

      setError(nextError);
      setHealth(healthResult.status === 'fulfilled' ? 'Terhubung' : 'Tidak terhubung');
    } finally {
      if (requestId === requestSeqRef.current) {
        setLoading(false);
      }
    }
  }, [isAdmin]);

  useEffect(() => {
    const timer = setTimeout(() => {
      void loadDashboard();
    }, 0);
    return () => clearTimeout(timer);
  }, [loadDashboard]);

  const { subscribe } = useWebSocket();

  const scheduleRefresh = useCallback(() => {
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current);
    }
    refreshTimerRef.current = setTimeout(() => {
      refreshTimerRef.current = null;
      void loadDashboard({ silent: true });
    }, REALTIME_REFRESH_DEBOUNCE_MS);
  }, [loadDashboard]);

  const applyLiveStatusUpdate = useCallback((payload?: CCTVLiveStatusPayload) => {
    if (!payload?.cctv_id) {
      scheduleRefresh();
      return;
    }

    const cctvId = payload.cctv_id;
    // Event dari site lain tidak ada di daftar; abaikan agar tidak memicu
    // refresh dashboard yang tidak relevan.
    if (!cctvsRef.current.some((item) => item.id === cctvId)) return;

    const update: Partial<CCTV> = {};
    if (payload.nama !== undefined) update.nama = payload.nama;
    if (payload.kode_cctv !== undefined) update.kode_cctv = payload.kode_cctv;
    if (payload.stream_live !== undefined) update.stream_live = payload.stream_live;
    if (payload.stream_state !== undefined) update.stream_state = payload.stream_state;
    if (payload.stream_last_active_at !== undefined) update.stream_last_active_at = payload.stream_last_active_at;
    if (payload.network_type !== undefined) update.network_type = payload.network_type;
    if (payload.stream_verified_state !== undefined) update.stream_verified_state = payload.stream_verified_state;
    if (payload.stream_verified_at !== undefined) update.stream_verified_at = payload.stream_verified_at;

    if (Object.keys(update).length === 0) return;

    setCctvs((current) => current.map((item) => (item.id === cctvId ? { ...item, ...update } : item)));
  }, [scheduleRefresh]);

  useEffect(() => {
    const unsubCCTV = subscribe('CCTV_UPDATED', scheduleRefresh);
    const unsubCCTVLive = subscribe('CCTV_LIVE_STATUS_UPDATED', applyLiveStatusUpdate);
    const unsubMaintCreated = subscribe('MAINTENANCE_CREATED', scheduleRefresh);
    const unsubMaintUpdated = subscribe('MAINTENANCE_UPDATED', scheduleRefresh);
    const unsubStockLow = subscribe('STOCK_LOW', () => {
      if (isAdmin) scheduleRefresh();
    });

    return () => {
      unsubCCTV();
      unsubCCTVLive();
      unsubMaintCreated();
      unsubMaintUpdated();
      unsubStockLow();
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current);
        refreshTimerRef.current = null;
      }
    };
  }, [subscribe, scheduleRefresh, applyLiveStatusUpdate, isAdmin]);

  useEffect(() => {
    cctvsRef.current = cctvs;
  }, [cctvs]);

  const liveCount = cctvs.filter((item) => getEffectiveStreamState(item) === 'live').length;
  const idleCount = cctvs.filter((item) => getEffectiveStreamState(item) === 'idle').length;
  const offlineCount = cctvs.filter((item) => getEffectiveStreamState(item) === 'offline').length;
  const maintenanceCount = cctvs.filter((item) => ['MAINTENANCE', 'PEMELIHARAAN'].includes(normalizeStatus(item.status))).length;

  const scheduled = logs.filter((item) => item.status === 'scheduled').length;
  const ongoing = logs.filter((item) => item.status === 'ongoing').length;
  const completed = logs.filter((item) => item.status === 'completed').length;
  const failed = logs.filter((item) => item.status === 'failed').length;
  const selectedLogs = selectedStatus ? logs.filter((item) => item.status === selectedStatus) : [];

  const summary: Array<{ key: CCTVFilter; label: string; value: number; tone: string }> = [
    { key: 'all', label: 'Total CCTV', value: totalCctv, tone: 'bg-blue-500' },
    { key: 'live', label: 'Online', value: liveCount, tone: 'bg-emerald-500' },
    { key: 'idle', label: 'Siaga', value: idleCount, tone: 'bg-blue-400' },
    { key: 'offline', label: 'Offline', value: offlineCount, tone: 'bg-rose-500' },
    { key: 'maintenance', label: 'Pemeliharaan', value: maintenanceCount, tone: 'bg-amber-500' },
  ];

  const selectedCctvs = (() => {
    switch (selectedCctvFilter) {
      case 'live':
        return cctvs.filter((item) => getEffectiveStreamState(item) === 'live');
      case 'idle':
        return cctvs.filter((item) => getEffectiveStreamState(item) === 'idle');
      case 'offline':
        return cctvs.filter((item) => getEffectiveStreamState(item) === 'offline');
      case 'maintenance':
        return cctvs.filter((item) => ['MAINTENANCE', 'PEMELIHARAAN'].includes(normalizeStatus(item.status)));
      case 'all':
        return cctvs;
      default:
        return [];
    }
  })();
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
          <div className="flex flex-wrap items-center gap-3">
            {user?.role === 'teknisi' && (
              <Link
                href={{ pathname: '/cctv', query: { scan: '1' } }}
                aria-label="Pindai QR CCTV untuk pemeliharaan"
                title="Pindai QR CCTV untuk pemeliharaan"
                className="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl border border-border bg-card py-1.5 pl-1.5 pr-3.5 text-sm font-bold text-foreground transition-colors hover:border-primary hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              >
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent text-[#203f25]">
                  <ScanLine className="h-5.5 w-5.5" strokeWidth={2.5} aria-hidden="true" />
                </span>
                Pindai
              </Link>
            )}
            <NotificationBell />
            <div className="flex items-center gap-2 text-xs font-semibold">
              <span className={`h-2 w-2 rounded-full ${health === 'Terhubung' ? 'bg-success' : health === 'Memeriksa' ? 'animate-pulse bg-warning' : 'bg-error'}`} />
              <span>{health === 'Terhubung' ? 'Layanan normal' : health}</span>
            </div>
            <div className="hidden md:block">
              <ProfileMenu />
            </div>
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
              <div className="grid divide-y divide-border sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 sm:divide-x sm:divide-y-0">
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
                      const streamState = getEffectiveStreamState(cctv);
                      const liveBadge = getLiveBadge(cctv);
                      const isMaintenance = ['MAINTENANCE', 'PEMELIHARAAN'].includes(normalizeStatus(cctv.status));
                      const isAnomalousOffline = !isMaintenance && streamState === 'offline';
                      const isAnomalousRecovered = isMaintenance && streamState === 'live';

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
                          <div className="flex flex-wrap items-center gap-2 sm:justify-end">
                            <span
                              className="inline-flex w-fit items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold"
                              style={{
                                borderColor: `${liveBadge.color}40`,
                                backgroundColor: `${liveBadge.color}15`,
                                color: liveBadge.color,
                              }}
                            >
                              <span>{liveBadge.dot}</span>
                              <span>{liveBadge.label}</span>
                            </span>

                            {isAnomalousOffline && (
                              <span
                                className="inline-flex items-center gap-1 rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[10px] font-bold text-amber-600 dark:text-amber-400"
                                title="Kamera terdaftar aktif tetapi sinyal RTSP offline"
                              >
                                ⚠️ Cek fisik
                              </span>
                            )}
                            {isAnomalousRecovered && (
                              <span
                                className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-bold text-emerald-600 dark:text-emerald-400"
                                title="Kamera dalam pemeliharaan tetapi sinyal live terdeteksi online"
                              >
                                🟢 Sinyal pulih
                              </span>
                            )}
                            {isMaintenance && !isAnomalousRecovered && (
                              <span className="inline-flex items-center rounded-full border border-amber-500/25 bg-amber-500/10 px-2 py-0.5 text-[10px] font-semibold text-amber-600 dark:text-amber-400">
                                Pemeliharaan
                              </span>
                            )}
                          </div>
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

            {/* Analitik: tren pemeliharaan + tugas berjalan, peringatan overdue */}
            <DashboardAnalytics summary={analytics} loading={loading}>
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
                        : 'Pekerjaan terjadwal untuk Anda.'}
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
                        <article key={task.id} className="grid gap-4 px-5 py-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                          <div className="min-w-0">
                            <p className="text-[11px] font-bold text-primary">Nama tugas</p>
                            <h3 className="mt-1 text-sm font-bold leading-snug" title={task.action_taken || undefined}>
                              {task.action_taken || 'Pemeliharaan CCTV'}
                            </h3>
                            <p className="mt-1.5 text-xs font-semibold text-foreground">
                              {task.cctv_name || 'CCTV tidak tersedia'}
                              {task.cctv_kode ? <span className="ml-1.5 font-mono text-primary">· {task.cctv_kode}</span> : null}
                            </p>
                            <p className="mt-1 flex items-center gap-1.5 text-xs text-muted">
                              <MapPin className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                              <span className="truncate" title={task.cctv_location || undefined}>
                                {task.cctv_location || 'Lokasi belum dicatat'}
                              </span>
                            </p>
                          </div>
                          <div className="flex flex-col items-start gap-2 sm:items-end">
                            <div className="flex flex-wrap items-center gap-2 sm:justify-end">
                              <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1 text-[11px] font-semibold text-muted">
                                <CalendarClock className="h-3.5 w-3.5" aria-hidden="true" />
                                {formatTaskSchedule(task.scheduled_date)}
                              </span>
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
                            {!isAdmin && task.status === 'ongoing' && (
                              <button
                                type="button"
                                onClick={() => setCompleteTask(task)}
                                className="inline-flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1.5 text-[11px] font-bold text-emerald-600 transition-colors hover:bg-emerald-500/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/40"
                              >
                                <CheckCheck className="h-3.5 w-3.5" />
                                Selesaikan
                              </button>
                            )}
                          </div>
                        </article>
                      );
                    })}
                  </div>
                )}
              </section>
            </DashboardAnalytics>

            {/* Widget suku cadang stok rendah — admin only (Task 11.1) */}
            {isAdmin && (
              <section aria-labelledby="low-stock-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
                <div className="flex flex-col gap-3 border-b border-border px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <h2 id="low-stock-heading" className="text-base font-bold">Suku Cadang Stok Rendah</h2>
                    <p className="mt-0.5 text-xs text-muted">Komponen yang perlu segera di-restock.</p>
                  </div>
                  <Link
                    href="/suku-cadang"
                    className="inline-flex items-center gap-1.5 rounded-lg border border-primary/30 bg-primary/10 px-3 py-1.5 text-xs font-bold text-primary transition-colors hover:bg-primary/20"
                  >
                    <Package className="h-3.5 w-3.5" />
                    Kelola Suku Cadang
                  </Link>
                </div>

                {lowStock.length === 0 ? (
                  <div className="flex items-center gap-3 px-5 py-6 text-sm text-muted">
                    <Package className="h-5 w-5 text-success" />
                    Semua suku cadang dalam stok yang cukup.
                  </div>
                ) : (
                  <div className="divide-y divide-border">
                    {lowStock.slice(0, 5).map((item) => (
                      <div
                        key={item.id}
                        className="grid gap-2 px-5 py-3.5 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:gap-4"
                      >
                        <div className="min-w-0">
                          <p className="truncate text-sm font-semibold" title={item.nama}>{item.nama}</p>
                          <p className="mt-0.5 font-mono text-xs text-primary">{item.sku}</p>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className="inline-flex items-center gap-1 rounded-full border border-error/25 bg-error/10 px-2.5 py-1 text-[11px] font-bold text-error">
                            <AlertTriangle className="h-3 w-3" />
                            {item.stok} {item.satuan}
                          </span>
                          <span className="text-[11px] text-muted">min. {item.minimum_order}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
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
                <StatusCard label="Perlu Tindak Lanjut" value={failed} tone="error" active={selectedStatus === 'failed'} onClick={() => setSelectedStatus((current) => current === 'failed' ? null : 'failed')} />
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

      {completeTask && (
        <FieldMaintenanceModal
          mode="complete"
          maintenanceLogId={completeTask.id}
          cctv={{
            id: completeTask.cctv_id,
            kode_cctv: completeTask.cctv_kode,
            nama: completeTask.cctv_name,
            lokasi_area: completeTask.cctv_location,
          }}
          open
          onClose={() => setCompleteTask(null)}
          onSaved={() => loadDashboard()}
        />
      )}
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
