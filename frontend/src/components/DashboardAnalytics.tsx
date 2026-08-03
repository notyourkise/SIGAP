'use client';

import TrendChart, { type TrendPoint } from './TrendChart';

interface OverdueItem {
  id: string;
  cctv_name: string;
  cctv_location: string;
  status: string;
  action_taken: string;
  scheduled_date: string;
  days_overdue: number;
}

export interface DashboardSummary {
  maintenance_trend: TrendPoint[];
  overdue: OverdueItem[];
}

const STATUS_LABEL: Record<string, { label: string; dot: string; bar: string }> = {
  scheduled: { label: 'Terjadwal', dot: 'bg-blue-500', bar: 'bg-blue-500' },
  ongoing: { label: 'Dalam proses', dot: 'bg-warning', bar: 'bg-warning' },
  completed: { label: 'Selesai', dot: 'bg-success', bar: 'bg-success' },
  failed: { label: 'Gagal', dot: 'bg-error', bar: 'bg-error' },
};

export default function DashboardAnalytics({
  summary,
  loading,
  children,
}: {
  summary: DashboardSummary | null;
  loading: boolean;
  children?: React.ReactNode;
}) {
  if (loading) {
    return (
      <div className="grid gap-4 lg:grid-cols-2" aria-label="Memuat analitik">
        <div className="h-64 animate-pulse rounded-2xl border border-border bg-card" />
        <div className="h-64 animate-pulse rounded-2xl border border-border bg-card" />
      </div>
    );
  }

  if (!summary) return null;

  const trend = summary.maintenance_trend ?? [];
  const overdue = summary.overdue ?? [];

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {/* Tren Pemeliharaan */}
      <section aria-labelledby="trend-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
        <div className="border-b border-border px-5 py-4">
          <h2 id="trend-heading" className="text-base font-bold">Tren Pemeliharaan</h2>
          <p className="mt-0.5 text-xs text-muted">Total pekerjaan per bulan — 6 bulan terakhir.</p>
        </div>

        {trend.length === 0 || trend.every((p) => p.scheduled + p.ongoing + p.completed + p.failed === 0) ? (
          <p className="px-5 py-8 text-sm text-muted">Belum ada data pemeliharaan dalam 6 bulan terakhir.</p>
        ) : (
          <>
            <div className="px-5 pt-5">
              <TrendChart data={trend} />
            </div>

            <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-border px-5 py-3.5">
              <span className="flex items-center gap-1.5 text-xs font-semibold text-muted">
                <span className="h-2.5 w-2.5 rounded-full bg-blue-500" /> Total pekerjaan
              </span>
              <span className="text-xs text-muted">Arahkan kursor ke titik untuk rincian per status.</span>
            </div>
          </>
        )}
      </section>

      {/* Slot pengganti Distribusi Status CCTV (card Tugas) */}
      {children}

      {/* Peringatan Overdue */}
      {overdue.length > 0 && (
        <section aria-labelledby="overdue-heading" className="overflow-hidden rounded-2xl border border-border bg-card lg:col-span-2">
          <div className="flex items-center justify-between border-b border-border px-5 py-4">
            <div>
              <h2 id="overdue-heading" className="text-base font-bold">Peringatan Pemeliharaan Terlambat</h2>
              <p className="mt-0.5 text-xs text-muted">Pekerjaan yang melewati jadwal wajib ditindaklanjuti.</p>
            </div>
            <span className="rounded-full border border-error/25 bg-error/10 px-2.5 py-1 text-xs font-bold text-error">{overdue.length}</span>
          </div>
          <div className="divide-y divide-border">
            {overdue.map((item) => (
              <div key={item.id} className="grid gap-2 px-5 py-4 sm:grid-cols-[minmax(0,1fr)_minmax(180px,0.7fr)_auto] sm:items-center sm:gap-4">
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold">{item.cctv_name || 'CCTV tidak tersedia'}</p>
                  <p className="mt-0.5 truncate text-xs text-muted">{item.action_taken}</p>
                </div>
                <p className="truncate text-xs text-muted" title={item.cctv_location || undefined}>
                  {item.cctv_location || 'Lokasi belum dicatat'}
                </p>
                <div className="flex items-center gap-2 sm:justify-end">
                  <span className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold ${
                    item.status === 'ongoing' ? 'border-warning/25 bg-warning/10 text-warning' : 'border-blue-500/25 bg-blue-500/10 text-blue-600 dark:text-blue-400'
                  }`}>
                    <span className={`h-1.5 w-1.5 rounded-full ${item.status === 'ongoing' ? 'bg-warning' : 'bg-blue-500'}`} />
                    {STATUS_LABEL[item.status]?.label ?? item.status}
                  </span>
                  <span className="rounded-full border border-error/25 bg-error/10 px-2.5 py-1 text-xs font-bold text-error">
                    Terlambat {Math.ceil(item.days_overdue)} hari
                  </span>
                </div>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
