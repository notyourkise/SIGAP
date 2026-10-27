'use client';

import { Suspense, useCallback, useEffect, useRef, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { useWebSocket } from '@/context/WebSocketContext';
import { apiRequest, getCookie } from '@/lib/api';
import {
  Activity,
  AlertTriangle,
  Clock,
  LoaderCircle,
  MapPin,
  Radio,
  RotateCw,
  Search,
  ShieldCheck,
  Wifi,
  WifiOff,
  X,
} from 'lucide-react';

interface CCTVInfo {
  id: string;
  nama: string;
  kode_cctv: string;
  lokasi_area?: string;
  ip_camera?: string;
  status: string;
}

interface CCTVStreamLogItem {
  id: string;
  created_at: string;
  cctv_id: string;
  cctv?: CCTVInfo;
  event_type: 'online' | 'offline' | 'standby' | 'unknown';
  previous_state: string;
  current_state: string;
  trigger_source: string;
  network_type: string;
  latency_ms?: number | null;
  error_detail: string;
  message: string;
}

interface LogSummary {
  today_online_events: number;
  today_offline_events: number;
  total_events: number;
  timezone?: string;
}

interface LogResponseData {
  rows: CCTVStreamLogItem[];
  pagination: {
    page: number;
    limit: number;
    total_rows: number;
    total_pages: number;
  };
  summary: LogSummary;
}

type EventTypeFilter = 'all' | 'online' | 'offline' | 'standby' | 'unknown';

interface QueryUpdate {
  event_type?: EventTypeFilter;
  search?: string;
  page?: number;
}

const LOG_LIMIT = 15;
const SEARCH_DEBOUNCE_MS = 300;

const EVENT_TYPE_FILTERS: ReadonlyArray<{ value: EventTypeFilter; label: string }> = [
  { value: 'all', label: 'Semua Event' },
  { value: 'online', label: 'Online' },
  { value: 'offline', label: 'Offline' },
  { value: 'standby', label: 'Siaga' },
  { value: 'unknown', label: 'Tidak Diketahui' },
];

const formatDateTime = (isoString: string) => {
  if (!isoString) return '-';
  const date = new Date(isoString);
  if (Number.isNaN(date.getTime())) return isoString;
  return new Intl.DateTimeFormat('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date);
};

const formatTimeAgo = (isoString: string) => {
  if (!isoString) return '';
  const date = new Date(isoString);
  if (Number.isNaN(date.getTime())) return '';
  const diffMs = Date.now() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return 'Baru saja';
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin} mnt lalu`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour} jam lalu`;
  const diffDay = Math.floor(diffHour / 24);
  return `${diffDay} hari lalu`;
};

const parseEventTypeFilter = (value: string | null): EventTypeFilter => {
  const match = EVENT_TYPE_FILTERS.find((option) => option.value === value);
  return match ? match.value : 'all';
};

const parsePage = (value: string | null): number => {
  const parsed = Number.parseInt(value ?? '', 10);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
};

function CCTVLogsContent() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  // URL menjadi sumber kebenaran filter agar tautan dapat dibagikan dan tetap
  // bertahan setelah refresh/back-forward.
  const page = parsePage(searchParams.get('page'));
  const eventTypeFilter = parseEventTypeFilter(searchParams.get('event_type'));
  const urlSearch = searchParams.get('search') ?? '';

  const [searchTerm, setSearchTerm] = useState(urlSearch);
  const [logs, setLogs] = useState<CCTVStreamLogItem[]>([]);
  const [summary, setSummary] = useState<LogSummary>({
    today_online_events: 0,
    today_offline_events: 0,
    total_events: 0,
    timezone: 'Asia/Makassar',
  });
  const [totalRows, setTotalRows] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [error, setError] = useState('');
  const requestIdRef = useRef(0);

  const updateQuery = useCallback(
    (updates: QueryUpdate) => {
      const params = new URLSearchParams(searchParams.toString());

      if (updates.event_type !== undefined) {
        if (updates.event_type === 'all') {
          params.delete('event_type');
        } else {
          params.set('event_type', updates.event_type);
        }
      }

      if (updates.search !== undefined) {
        const trimmed = updates.search.trim();
        if (trimmed) {
          params.set('search', trimmed);
        } else {
          params.delete('search');
        }
      }

      if (updates.page !== undefined) {
        if (updates.page > 1) {
          params.set('page', String(updates.page));
        } else {
          params.delete('page');
        }
      }

      const query = params.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
    },
    [pathname, router, searchParams]
  );

  // Sinkronkan input saat URL berubah dari luar (mis. tombol back browser atau
  // tautan yang dibagikan).
  useEffect(() => {
    setSearchTerm(urlSearch);
  }, [urlSearch]);

  // Debounce 300 ms: URL (dan karena itu fetch) baru berubah setelah pengguna
  // berhenti mengetik, sehingga setiap ketikan tidak membanjiri API.
  useEffect(() => {
    if (searchTerm.trim() === urlSearch.trim()) return;

    const timer = window.setTimeout(() => {
      updateQuery({ search: searchTerm, page: 1 });
    }, SEARCH_DEBOUNCE_MS);

    return () => window.clearTimeout(timer);
  }, [searchTerm, urlSearch, updateQuery]);

  const fetchLogs = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    setError('');

    try {
      const params = new URLSearchParams({
        page: String(page),
        limit: String(LOG_LIMIT),
      });
      if (eventTypeFilter !== 'all') {
        params.set('event_type', eventTypeFilter);
      }
      if (urlSearch.trim()) {
        params.set('search', urlSearch.trim());
      }

      const res = await apiRequest<LogResponseData>(`/cctv-logs?${params.toString()}`);

      // Abaikan respons lama yang datang setelah permintaan yang lebih baru.
      if (requestId !== requestIdRef.current) return;

      if (res && res.data) {
        setLogs(res.data.rows || []);
        setTotalRows(res.data.pagination?.total_rows ?? 0);
        setTotalPages(res.data.pagination?.total_pages ?? 1);
        if (res.data.summary) {
          setSummary(res.data.summary);
        }
      }
      setHasLoaded(true);
      setLastUpdated(new Date());
    } catch (err) {
      if (requestId !== requestIdRef.current) return;
      setError(err instanceof Error ? err.message : 'Gagal memuat log status CCTV');
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, [eventTypeFilter, page, urlSearch]);

  useEffect(() => {
    void fetchLogs();
  }, [fetchLogs]);

  // Event WebSocket hanya memicu refetch bila berasal dari site aktif.
  // Backend sudah memfilter per penugasan; filter ini mencegah refetch saat
  // admin sedang memilih satu site namun menerima event dari site lain.
  const shouldHandleSiteEvent = useCallback((payload: unknown) => {
    const activeSiteId = getCookie('sigap_site_id');
    if (!activeSiteId) return true;

    const eventSiteId =
      payload && typeof payload === 'object' && 'site_id' in payload
        ? (payload as { site_id?: string | null }).site_id
        : undefined;
    return !eventSiteId || eventSiteId === activeSiteId;
  }, []);

  // Real-time synchronization via WebSocket
  const { subscribe } = useWebSocket();
  useEffect(() => {
    const unsubLog = subscribe('CCTV_STREAM_LOG_CREATED', (payload) => {
      if (shouldHandleSiteEvent(payload)) void fetchLogs();
    });
    const unsubLive = subscribe('CCTV_LIVE_STATUS_UPDATED', (payload) => {
      if (shouldHandleSiteEvent(payload)) void fetchLogs();
    });

    return () => {
      unsubLog();
      unsubLive();
    };
  }, [subscribe, fetchLogs, shouldHandleSiteEvent]);

  const resetFilters = useCallback(() => {
    setSearchTerm('');
    updateQuery({ event_type: 'all', search: '', page: 1 });
  }, [updateQuery]);

  const hasActiveFilters = eventTypeFilter !== 'all' || urlSearch.trim() !== '';

  const eventBadge = (type: string) => {
    switch (type) {
      case 'online':
        return {
          label: 'Online',
          icon: Wifi,
          className: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
          dot: 'bg-emerald-500',
        };
      case 'offline':
        return {
          label: 'Offline',
          icon: WifiOff,
          className: 'border-rose-500/25 bg-rose-500/10 text-rose-600 dark:text-rose-400',
          dot: 'bg-rose-500',
        };
      case 'standby':
        return {
          label: 'Siaga',
          icon: Radio,
          className: 'border-blue-500/25 bg-blue-500/10 text-blue-600 dark:text-blue-400',
          dot: 'bg-blue-500',
        };
      default:
        return {
          label: 'Tidak Diketahui',
          icon: AlertTriangle,
          className: 'border-muted/30 bg-muted/10 text-muted',
          dot: 'bg-muted',
        };
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        {/* Header */}
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Activity className="h-5 w-5" />
              </span>
              <h1 className="text-3xl font-bold tracking-tight">Log Status CCTV</h1>
            </div>
            <p className="mt-1 text-sm text-muted">
              Riwayat kronologis deteksi event Online/Offline dan hasil verifikasi stream seluruh kamera secara real-time.
            </p>
          </div>

          <div className="flex flex-col items-start gap-1.5 sm:items-end">
            <button
              type="button"
              onClick={() => void fetchLogs()}
              disabled={loading}
              className="inline-flex items-center gap-1.5 rounded-xl border border-border bg-card px-3.5 py-2 text-xs font-semibold text-foreground transition-colors hover:bg-background disabled:opacity-50"
              title="Perbarui log"
            >
              <RotateCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              Segarkan
            </button>
            <p className="text-[11px] text-muted" aria-live="polite">
              {loading ? (
                <span className="inline-flex items-center gap-1">
                  <LoaderCircle className="h-3 w-3 animate-spin" />
                  {hasLoaded ? 'Memperbarui…' : 'Memuat data…'}
                </span>
              ) : lastUpdated ? (
                <>Terakhir diperbarui {formatDateTime(lastUpdated.toISOString())}</>
              ) : (
                'Belum pernah diperbarui'
              )}
            </p>
          </div>
        </header>

        {/* Metric Summary Cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="flex items-center justify-between rounded-2xl border border-border bg-card p-5">
            <div>
              <p className="text-xs font-semibold text-muted">Total hasil filter</p>
              <p className="mt-1 text-2xl font-bold tabular-nums text-foreground">{summary.total_events}</p>
            </div>
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-blue-500/10 text-blue-500">
              <Clock className="h-5 w-5" />
            </div>
          </div>

          <div className="flex items-center justify-between rounded-2xl border border-border bg-card p-5">
            <div>
              <p className="text-xs font-semibold text-muted">Event online hari ini</p>
              <p className="mt-1 text-2xl font-bold tabular-nums text-emerald-600 dark:text-emerald-400">
                {summary.today_online_events}
              </p>
            </div>
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-500">
              <ShieldCheck className="h-5 w-5" />
            </div>
          </div>

          <div className="flex items-center justify-between rounded-2xl border border-border bg-card p-5">
            <div>
              <p className="text-xs font-semibold text-muted">Event offline hari ini</p>
              <p className="mt-1 text-2xl font-bold tabular-nums text-rose-600 dark:text-rose-400">
                {summary.today_offline_events}
              </p>
            </div>
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-rose-500/10 text-rose-500">
              <AlertTriangle className="h-5 w-5" />
            </div>
          </div>
        </div>

        <p className="-mt-3 text-[11px] text-muted">
          Kartu event hari ini mengikuti site/filter aktif. Batas hari dihitung dalam zona waktu{' '}
          {summary.timezone === 'Asia/Makassar' ? 'WITA' : summary.timezone || 'WITA'}.
        </p>

        {/* Filter and Search Bar */}
        <div className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative w-full max-w-md flex-1">
            <Search className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
            <input
              type="text"
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
              placeholder="Cari kode atau nama CCTV (mis. THD-12)..."
              aria-label="Cari log berdasarkan kode atau nama CCTV"
              className="w-full rounded-xl border border-border bg-background py-2 pl-10 pr-10 text-xs font-medium placeholder:text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
            {searchTerm && (
              <button
                type="button"
                onClick={() => {
                  setSearchTerm('');
                  updateQuery({ search: '', page: 1 });
                }}
                aria-label="Hapus pencarian"
                className="absolute right-2 top-1/2 flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-lg text-muted transition-colors hover:bg-background hover:text-foreground"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            {EVENT_TYPE_FILTERS.map((option) => {
              const active = eventTypeFilter === option.value;
              return (
                <button
                  key={option.value}
                  type="button"
                  aria-pressed={active}
                  onClick={() => updateQuery({ event_type: option.value, page: 1 })}
                  className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors ${
                    active
                      ? 'bg-primary text-white'
                      : 'border border-border bg-background text-muted hover:bg-card hover:text-foreground'
                  }`}
                >
                  {option.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Error banner tetap menampilkan data lama agar pengguna tidak
            kehilangan konteks saat refresh gagal. */}
        {error && logs.length > 0 && (
          <div
            role="alert"
            className="flex flex-col gap-3 rounded-xl border border-rose-500/30 bg-rose-500/10 p-4 text-xs text-rose-600 dark:text-rose-400 sm:flex-row sm:items-center sm:justify-between"
          >
            <div className="flex items-start gap-2">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <div>
                <p className="font-bold">Gagal memperbarui data</p>
                <p className="mt-0.5">{error}</p>
                <p className="mt-1 text-[11px] opacity-80">Data terakhir tetap ditampilkan.</p>
              </div>
            </div>
            <button
              type="button"
              onClick={() => void fetchLogs()}
              disabled={loading}
              className="inline-flex shrink-0 items-center justify-center gap-1.5 rounded-lg border border-rose-500/40 bg-card px-3 py-2 font-semibold text-rose-600 transition-colors hover:bg-rose-500/10 disabled:opacity-50 dark:text-rose-400"
            >
              <RotateCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              Coba lagi
            </button>
          </div>
        )}

        {/* Log Timeline Table */}
        <section aria-label="Daftar Log Event CCTV" className="overflow-hidden rounded-2xl border border-border bg-card">
          {loading && !hasLoaded ? (
            <div className="space-y-4 p-6">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="h-14 animate-pulse rounded-xl bg-background" />
              ))}
            </div>
          ) : error && logs.length === 0 ? (
            <div className="flex flex-col items-center justify-center p-12 text-center">
              <AlertTriangle className="h-10 w-10 text-rose-500 opacity-70" />
              <p className="mt-3 text-sm font-bold text-foreground">Gagal memuat log status</p>
              <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
              <button
                type="button"
                onClick={() => void fetchLogs()}
                disabled={loading}
                className="mt-4 inline-flex items-center gap-1.5 rounded-xl border border-border bg-background px-4 py-2 text-xs font-semibold text-foreground transition-colors hover:bg-card disabled:opacity-50"
              >
                <RotateCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
                Coba lagi
              </button>
            </div>
          ) : logs.length === 0 ? (
            hasActiveFilters ? (
              <div className="flex flex-col items-center justify-center p-12 text-center">
                <Search className="h-10 w-10 text-muted opacity-40" />
                <p className="mt-3 text-sm font-bold text-foreground">Tidak ada hasil filter</p>
                <p className="mt-1 max-w-sm text-xs text-muted">
                  {urlSearch.trim()
                    ? `Tidak ada log yang cocok dengan pencarian “${urlSearch.trim()}” pada filter ini.`
                    : 'Tidak ada log yang cocok dengan filter event yang dipilih.'}
                </p>
                <button
                  type="button"
                  onClick={resetFilters}
                  className="mt-4 inline-flex items-center gap-1.5 rounded-xl border border-border bg-background px-4 py-2 text-xs font-semibold text-foreground transition-colors hover:bg-card"
                >
                  <X className="h-3.5 w-3.5" />
                  Hapus filter
                </button>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center p-12 text-center">
                <Activity className="h-10 w-10 text-muted opacity-40" />
                <p className="mt-3 text-sm font-bold text-foreground">Belum ada catatan log status</p>
                <p className="mt-1 max-w-sm text-xs text-muted">
                  Setiap kali engine probe atau verifier mendeteksi kamera beralih Online atau Offline, riwayatnya akan tersimpan di sini.
                </p>
              </div>
            )
          ) : (
            <div className="divide-y divide-border">
              {logs.map((item) => {
                const badge = eventBadge(item.event_type);
                const BadgeIcon = badge.icon;
                return (
                  <article
                    key={item.id}
                    className="grid gap-3 p-4 transition-colors hover:bg-background/60 sm:grid-cols-[160px_minmax(0,1.2fr)_auto] sm:items-center sm:gap-6"
                  >
                    {/* Timestamp & TimeAgo */}
                    <div className="min-w-0">
                      <p className="text-xs font-bold text-foreground">{formatDateTime(item.created_at)}</p>
                      <p className="mt-0.5 text-[11px] font-medium text-muted">{formatTimeAgo(item.created_at)}</p>
                    </div>

                    {/* CCTV Identity & Details */}
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-mono text-xs font-bold text-primary">
                          {item.cctv?.kode_cctv || 'CCTV'}
                        </span>
                        <h2 className="truncate text-sm font-semibold text-foreground">
                          {item.cctv?.nama || 'Perangkat CCTV'}
                        </h2>
                      </div>

                      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
                        {item.cctv?.lokasi_area && (
                          <span className="flex items-center gap-1">
                            <MapPin className="h-3 w-3 shrink-0" />
                            <span className="truncate">{item.cctv.lokasi_area}</span>
                          </span>
                        )}
                        <span className="capitalize">
                          Jaringan: <strong className="font-semibold text-foreground">{item.network_type}</strong>
                        </span>
                        <span>
                          Metode: <strong className="font-semibold text-foreground">{item.trigger_source === 'verify' ? 'Verifikasi HLS' : 'Probe Jaringan'}</strong>
                        </span>
                        {typeof item.latency_ms === 'number' && (
                          <span>
                            Latensi: <strong className="font-mono text-foreground">{item.latency_ms} ms</strong>
                          </span>
                        )}
                      </div>

                      {item.message && (
                        <p className="mt-1.5 text-xs text-muted line-clamp-2" title={item.message}>
                          {item.message}
                        </p>
                      )}
                    </div>

                    {/* Event Status Badge & State Transition */}
                    <div className="flex flex-col items-start gap-1 sm:items-end">
                      <span
                        className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-bold ${badge.className}`}
                      >
                        <BadgeIcon className="h-3.5 w-3.5" />
                        <span>{badge.label}</span>
                      </span>

                      {item.previous_state && item.current_state && item.previous_state !== item.current_state && (
                        <p className="font-mono text-[10px] text-muted">
                          {item.previous_state} → {item.current_state}
                        </p>
                      )}
                    </div>
                  </article>
                );
              })}
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="border-t border-border p-3">
              <Pagination
                page={page}
                limit={LOG_LIMIT}
                totalRows={totalRows}
                totalPages={totalPages}
                onPageChange={(nextPage) => updateQuery({ page: nextPage })}
                itemLabel="Catatan Log"
              />
            </div>
          )}
        </section>
      </div>
    </DashboardLayout>
  );
}

function CCTVLogsFallback() {
  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="h-20 animate-pulse rounded-2xl bg-card" />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="h-24 animate-pulse rounded-2xl bg-card" />
          ))}
        </div>
        <div className="h-16 animate-pulse rounded-2xl bg-card" />
        <div className="h-80 animate-pulse rounded-2xl bg-card" />
      </div>
    </DashboardLayout>
  );
}

export default function CCTVLogsPage() {
  return (
    <Suspense fallback={<CCTVLogsFallback />}>
      <CCTVLogsContent />
    </Suspense>
  );
}
