'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AlertTriangle, ArrowDown, ArrowDownUp, ArrowUp, BarChart3, LoaderCircle, TrendingUp } from 'lucide-react';
import DashboardLayout from '@/components/DashboardLayout';
import TechnicianPerformanceChart from '@/components/TechnicianPerformanceChart';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';

interface TechnicianPerformance {
  user_id: string;
  full_name: string;
  username: string;
  total_tasks: number;
  completed: number;
  failed: number;
  ongoing: number;
  completion_rate: number;
  avg_completion_hours: number;
  overdue_count: number;
  field_check_count: number;
  quality_score: number;
}

type RangePreset = '7' | '30' | 'custom';
type SortKey =
  | 'full_name'
  | 'total_tasks'
  | 'completed'
  | 'failed'
  | 'ongoing'
  | 'completion_rate'
  | 'avg_completion_hours'
  | 'overdue_count'
  | 'field_check_count'
  | 'quality_score';

const SORT_LABELS: Record<SortKey, string> = {
  full_name: 'Nama teknisi',
  total_tasks: 'Total tugas',
  completed: 'Selesai',
  failed: 'Gagal',
  ongoing: 'Ongoing',
  completion_rate: 'Completion rate',
  avg_completion_hours: 'Rata-rata waktu',
  overdue_count: 'Overdue',
  field_check_count: 'Pengecekan',
  quality_score: 'Kualitas',
};

function toDateInputValue(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function dateDaysAgo(days: number) {
  const date = new Date();
  date.setDate(date.getDate() - days);
  return toDateInputValue(date);
}

function formatDisplayDate(value: string) {
  if (!value) return '-';
  const date = new Date(`${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

function formatNumber(value: number, digits = 0) {
  return new Intl.NumberFormat('id-ID', {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(Number.isFinite(value) ? value : 0);
}

function rateBadgeClass(rate: number) {
  if (rate >= 80) return 'border-success/25 bg-success/10 text-success';
  if (rate >= 50) return 'border-warning/25 bg-warning/10 text-warning';
  return 'border-error/25 bg-error/10 text-error';
}

function qualityBadgeClass(row: TechnicianPerformance) {
  if (row.field_check_count === 0) return 'border-border bg-background/60 text-muted';
  return rateBadgeClass(row.quality_score);
}

function RateBadge({ value, className }: { value: number; className: string }) {
  return (
    <span className={`inline-flex min-w-16 justify-center rounded-full border px-2.5 py-1 text-xs font-bold tabular-nums ${className}`}>
      {formatNumber(value, 1)}%
    </span>
  );
}

function LoadingRows() {
  return (
    <div className="space-y-3 px-5 py-5" aria-label="Memuat data kinerja" role="status">
      {[1, 2, 3, 4].map((item) => (
        <div key={item} className="grid grid-cols-4 gap-3 animate-pulse sm:grid-cols-10">
          {Array.from({ length: 10 }, (_, index) => (
            <div key={index} className={`h-8 rounded-lg bg-background ${index === 0 ? 'col-span-2' : ''}`} />
          ))}
        </div>
      ))}
      <span className="sr-only">Memuat data kinerja teknisi...</span>
    </div>
  );
}

export default function TechnicianPerformancePage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
  const [range, setRange] = useState<RangePreset>('30');
  const [customStart, setCustomStart] = useState(dateDaysAgo(29));
  const [customEnd, setCustomEnd] = useState(toDateInputValue(new Date()));
  const [performance, setPerformance] = useState<TechnicianPerformance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [sortKey, setSortKey] = useState<SortKey>('completion_rate');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');

  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user?.role]);

  const loadPerformance = useCallback(async () => {
    if (user?.role !== 'admin') return;

    let startDate = customStart;
    let endDate = customEnd;
    if (range === '7') {
      startDate = dateDaysAgo(6);
      endDate = toDateInputValue(new Date());
    } else if (range === '30') {
      startDate = dateDaysAgo(29);
      endDate = toDateInputValue(new Date());
    }

    if (!startDate || !endDate || startDate > endDate) {
      setPerformance([]);
      setLoading(false);
      setError('Tanggal mulai harus sama atau sebelum tanggal akhir.');
      return;
    }

    setLoading(true);
    setError('');
    const params = new URLSearchParams({ start_date: startDate, end_date: endDate });

    try {
      const response = await apiRequest<TechnicianPerformance[]>(`/dashboard/technician-performance?${params.toString()}`);
      setPerformance(response.data || []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Data kinerja teknisi belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [customEnd, customStart, range, user?.role]);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      await Promise.resolve();
      if (!cancelled) await loadPerformance();
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [loadPerformance]);

  const sortedPerformance = [...performance].sort((a, b) => {
    const aValue = a[sortKey];
    const bValue = b[sortKey];
    const result = typeof aValue === 'number' && typeof bValue === 'number'
      ? aValue - bValue
      : String(aValue).localeCompare(String(bValue), 'id');
    return sortDirection === 'asc' ? result : -result;
  });

  const chartData = [...performance]
    .sort((a, b) => b.completion_rate - a.completion_rate || a.full_name.localeCompare(b.full_name, 'id'))
    .map(({ user_id, full_name, completion_rate }) => ({ user_id, full_name, completion_rate }));

  const activeStart = range === '7'
    ? dateDaysAgo(6)
    : range === '30'
      ? dateDaysAgo(29)
      : customStart;
  const activeEnd = range === 'custom' ? customEnd : toDateInputValue(new Date());

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDirection((current) => current === 'asc' ? 'desc' : 'asc');
      return;
    }
    setSortKey(key);
    setSortDirection(key === 'full_name' ? 'asc' : 'desc');
  };

  const renderSortHeader = (key: SortKey) => {
    const isActive = sortKey === key;
    return (
      <button
        type="button"
        onClick={() => handleSort(key)}
        aria-label={`Urutkan berdasarkan ${SORT_LABELS[key]}`}
        className="inline-flex items-center gap-1.5 rounded-md text-left transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
      >
        {SORT_LABELS[key]}
        {isActive ? (
          sortDirection === 'asc' ? <ArrowUp className="h-3.5 w-3.5 text-primary" /> : <ArrowDown className="h-3.5 w-3.5 text-primary" />
        ) : (
          <ArrowDownUp className="h-3.5 w-3.5 text-muted/50" />
        )}
      </button>
    );
  };

  if (!authLoading && user?.role !== 'admin') return null;

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.08em] text-primary">
              <TrendingUp className="h-4 w-4" />
              <span>Evaluasi operasional</span>
            </div>
            <h1 className="mt-2 text-3xl font-bold tracking-[-0.025em]">Kinerja Teknisi</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Bandingkan beban tugas, kecepatan penyelesaian, dan kualitas pengecekan dalam satu rentang laporan.
            </p>
          </div>
          <div className="rounded-xl border border-border bg-card px-4 py-3 text-sm">
            <p className="text-xs font-semibold text-muted">Rentang aktif</p>
            <p className="mt-1 font-semibold">{formatDisplayDate(activeStart)} - {formatDisplayDate(activeEnd)}</p>
          </div>
        </header>

        <section aria-labelledby="performance-filter-heading" className="rounded-2xl border border-border bg-card p-4 sm:p-5">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 id="performance-filter-heading" className="font-semibold">Rentang laporan</h2>
              <p className="mt-1 text-xs text-muted">Data tugas memakai waktu penugasan; pengecekan memakai waktu pencatatan.</p>
            </div>
            <div className="flex flex-wrap gap-2" role="group" aria-label="Pilihan rentang waktu">
              {([
                ['7', '7 Hari'],
                ['30', '30 Hari'],
                ['custom', 'Kustom'],
              ] as const).map(([value, label]) => (
                <button
                  key={value}
                  type="button"
                  aria-pressed={range === value}
                  onClick={() => setRange(value)}
                  className={`min-h-11 rounded-xl border px-4 py-2.5 text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 ${range === value ? 'border-primary bg-primary text-white shadow-[0_3px_10px_rgba(70,114,53,0.18)]' : 'border-border bg-background text-muted hover:border-primary/35 hover:text-foreground'}`}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          {range === 'custom' && (
            <div className="mt-4 grid gap-3 border-t border-border pt-4 sm:grid-cols-2 sm:max-w-xl">
              <label className="text-xs font-semibold text-muted">
                Mulai
                <input
                  type="date"
                  value={customStart}
                  onChange={(event) => setCustomStart(event.target.value)}
                  className="mt-1.5 block min-h-11 w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm font-normal text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                />
              </label>
              <label className="text-xs font-semibold text-muted">
                Sampai
                <input
                  type="date"
                  value={customEnd}
                  onChange={(event) => setCustomEnd(event.target.value)}
                  className="mt-1.5 block min-h-11 w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm font-normal text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                />
              </label>
            </div>
          )}
        </section>

        <section aria-labelledby="completion-chart-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h2 id="completion-chart-heading" className="text-lg font-bold tracking-[-0.015em]">Completion rate per teknisi</h2>
              <p className="mt-1 text-xs text-muted">Persentase tugas berstatus selesai dari seluruh tugas yang ditugaskan pada periode ini.</p>
            </div>
            <div className="flex flex-wrap gap-2 text-[11px] font-semibold text-muted">
              <span className="rounded-full border border-success/25 bg-success/10 px-2.5 py-1 text-success">80% ke atas</span>
              <span className="rounded-full border border-warning/25 bg-warning/10 px-2.5 py-1 text-warning">50-79%</span>
              <span className="rounded-full border border-error/25 bg-error/10 px-2.5 py-1 text-error">Di bawah 50%</span>
            </div>
          </div>
          <div className="mt-5">
            {loading ? (
              <div className="flex min-h-24 items-center justify-center gap-3 text-sm text-muted" role="status">
                <LoaderCircle className="h-5 w-5 animate-spin text-primary" />
                Memuat grafik...
              </div>
            ) : error ? (
              <div className="flex items-start gap-3 rounded-xl border border-warning/25 bg-warning/10 px-4 py-3 text-sm" role="alert">
                <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
                <span>{error}</span>
              </div>
            ) : chartData.length === 0 ? (
              <div className="flex min-h-24 items-center justify-center text-sm text-muted">Belum ada aktivitas teknisi pada rentang ini.</div>
            ) : (
              <TechnicianPerformanceChart data={chartData} />
            )}
          </div>
        </section>

        <section aria-labelledby="performance-table-heading" aria-busy={loading} className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex flex-col gap-1 border-b border-border px-5 py-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 id="performance-table-heading" className="font-semibold">Ranking kinerja</h2>
              <p className="mt-0.5 text-xs text-muted">{performance.length} teknisi dengan aktivitas pada rentang terpilih</p>
            </div>
            <p className="text-xs text-muted">Klik judul kolom untuk mengurutkan</p>
          </div>

          {loading ? (
            <LoadingRows />
          ) : error ? (
            <div className="flex flex-col items-center justify-center px-6 py-14 text-center" role="alert">
              <AlertTriangle className="h-9 w-9 text-error" />
              <h3 className="mt-3 font-semibold">Data kinerja belum dapat dimuat</h3>
              <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
              <button
                type="button"
                onClick={() => void loadPerformance()}
                className="mt-4 rounded-xl bg-primary px-4 py-2.5 text-xs font-bold text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              >
                Coba lagi
              </button>
            </div>
          ) : sortedPerformance.length === 0 ? (
            <div className="flex flex-col items-center justify-center px-6 py-14 text-center">
              <BarChart3 className="h-10 w-10 text-muted/45" />
              <h3 className="mt-3 font-semibold">Belum ada data kinerja</h3>
              <p className="mt-1 max-w-sm text-xs text-muted">Pilih rentang yang lebih panjang atau tunggu sampai ada tugas dan pengecekan yang tercatat.</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[1150px] border-collapse text-left text-sm">
                <thead className="bg-background/55 text-xs font-semibold text-muted">
                  <tr className="border-b border-border">
                    <th scope="col" aria-sort={sortKey === 'full_name' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-5 py-3.5">{renderSortHeader('full_name')}</th>
                    <th scope="col" aria-sort={sortKey === 'total_tasks' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('total_tasks')}</th>
                    <th scope="col" aria-sort={sortKey === 'completed' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('completed')}</th>
                    <th scope="col" aria-sort={sortKey === 'failed' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('failed')}</th>
                    <th scope="col" aria-sort={sortKey === 'ongoing' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('ongoing')}</th>
                    <th scope="col" aria-sort={sortKey === 'completion_rate' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('completion_rate')}</th>
                    <th scope="col" aria-sort={sortKey === 'avg_completion_hours' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('avg_completion_hours')}</th>
                    <th scope="col" aria-sort={sortKey === 'overdue_count' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('overdue_count')}</th>
                    <th scope="col" aria-sort={sortKey === 'field_check_count' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('field_check_count')}</th>
                    <th scope="col" aria-sort={sortKey === 'quality_score' ? sortDirection === 'asc' ? 'ascending' : 'descending' : 'none'} className="px-4 py-3.5">{renderSortHeader('quality_score')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {sortedPerformance.map((row, index) => (
                    <tr key={row.user_id} className="transition-colors hover:bg-background/40">
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-3">
                          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-xs font-bold text-primary">{index + 1}</span>
                          <div className="min-w-0">
                            <p className="truncate font-semibold" title={row.full_name}>{row.full_name}</p>
                            <p className="mt-0.5 text-xs text-muted">@{row.username}</p>
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-4 font-semibold tabular-nums">{formatNumber(row.total_tasks)}</td>
                      <td className="px-4 py-4 font-semibold tabular-nums text-success">{formatNumber(row.completed)}</td>
                      <td className="px-4 py-4 font-semibold tabular-nums text-error">{formatNumber(row.failed)}</td>
                      <td className="px-4 py-4 font-semibold tabular-nums text-warning">{formatNumber(row.ongoing)}</td>
                      <td className="px-4 py-4"><RateBadge value={row.completion_rate} className={rateBadgeClass(row.completion_rate)} /></td>
                      <td className="px-4 py-4 tabular-nums text-muted">{formatNumber(row.avg_completion_hours, 1)} jam</td>
                      <td className="px-4 py-4 font-semibold tabular-nums">{formatNumber(row.overdue_count)}</td>
                      <td className="px-4 py-4 tabular-nums">{formatNumber(row.field_check_count)}</td>
                      <td className="px-4 py-4">
                        {row.field_check_count === 0 ? (
                          <span className="text-xs text-muted">Belum ada</span>
                        ) : (
                          <RateBadge value={row.quality_score} className={qualityBadgeClass(row)} />
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </DashboardLayout>
  );
}
