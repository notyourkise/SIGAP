'use client';

interface StatusCount {
  status: string;
  count: number;
}

interface TrendPoint {
  month: string;
  scheduled: number;
  ongoing: number;
  completed: number;
  failed: number;
}

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
  cctv_status: StatusCount[];
  maintenance_trend: TrendPoint[];
  overdue: OverdueItem[];
}

const STATUS_LABEL: Record<string, { label: string; dot: string; bar: string }> = {
  scheduled: { label: 'Terjadwal', dot: 'bg-blue-500', bar: 'bg-blue-500' },
  ongoing: { label: 'Dalam proses', dot: 'bg-warning', bar: 'bg-warning' },
  completed: { label: 'Selesai', dot: 'bg-success', bar: 'bg-success' },
  failed: { label: 'Gagal', dot: 'bg-error', bar: 'bg-error' },
};

const cctvStatusLabel = (status: string) => {
  const map: Record<string, string> = { aktif: 'Aktif', rusak: 'Rusak', pemeliharaan: 'Pemeliharaan', nonaktif: 'Nonaktif' };
  return map[status] ?? (status || 'Belum ditentukan');
};

const formatMonth = (month: string) => {
  const [year, m] = month.split('-');
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
  return `${months[Number(m) - 1]} ${year.slice(2)}`;
};

export default function DashboardAnalytics({ summary, loading }: { summary: DashboardSummary | null; loading: boolean }) {
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
  const maxTotal = Math.max(1, ...trend.map((p) => p.scheduled + p.ongoing + p.completed + p.failed));
  const statuses = summary.cctv_status ?? [];
  const statusTotal = statuses.reduce((acc, s) => acc + s.count, 0);
  const overdue = summary.overdue ?? [];

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {/* Tren Pemeliharaan */}
      <section aria-labelledby="trend-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
        <div className="border-b border-border px-5 py-4">
          <h2 id="trend-heading" className="text-base font-bold">Tren Pemeliharaan</h2>
          <p className="mt-0.5 text-xs text-muted">Jumlah pekerjaan per status — 6 bulan terakhir.</p>
        </div>

        {trend.length === 0 || trend.every((p) => p.scheduled + p.ongoing + p.completed + p.failed === 0) ? (
          <p className="px-5 py-8 text-sm text-muted">Belum ada data pemeliharaan dalam 6 bulan terakhir.</p>
        ) : (
          <>
            <div className="flex items-end gap-2 px-5 pt-6">
              {trend.map((point) => {
                const total = point.scheduled + point.ongoing + point.completed + point.failed;
                const barHeight = total === 0 ? 4 : Math.max(6, Math.round((total / maxTotal) * 140));
                const segments = [
                  { value: point.scheduled, tone: STATUS_LABEL.scheduled.bar },
                  { value: point.ongoing, tone: STATUS_LABEL.ongoing.bar },
                  { value: point.completed, tone: STATUS_LABEL.completed.bar },
                  { value: point.failed, tone: STATUS_LABEL.failed.bar },
                ].filter((segment) => segment.value > 0);
                return (
                  <div key={point.month} className="group flex flex-1 flex-col items-center gap-2">
                    <div className="relative flex h-[140px] w-full items-end justify-center">
                      {total === 0 ? (
                        <div className="w-full max-w-9 rounded-md border border-dashed border-border" style={{ height: barHeight }} />
                      ) : (
                        <div
                          className="flex w-full max-w-9 flex-col-reverse overflow-hidden rounded-md"
                          style={{ height: barHeight }}
                          title={`${point.month}: Terjadwal ${point.scheduled} · Dalam proses ${point.ongoing} · Selesai ${point.completed} · Gagal ${point.failed}`}
                        >
                          {segments.map((segment, index) => (
                            <div
                              key={index}
                              className={segment.tone}
                              style={{ height: `${Math.max(2, (segment.value / total) * 100)}%` }}
                            />
                          ))}
                        </div>
                      )}
                      <span className="absolute -top-5 text-xs font-bold tabular-nums text-muted opacity-0 transition-opacity group-hover:opacity-100">
                        {total}
                      </span>
                    </div>
                    <span className="text-[11px] font-semibold text-muted">{formatMonth(point.month)}</span>
                  </div>
                );
              })}
            </div>

            <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-border px-5 py-3.5">
              {(['scheduled', 'ongoing', 'completed', 'failed'] as const).map((key) => (
                <span key={key} className="flex items-center gap-1.5 text-xs font-semibold text-muted">
                  <span className={`h-2 w-2 rounded-full ${STATUS_LABEL[key].dot}`} />
                  {STATUS_LABEL[key].label}
                </span>
              ))}
            </div>
          </>
        )}
      </section>

      {/* Distribusi Status CCTV */}
      <section aria-labelledby="cctv-status-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
        <div className="border-b border-border px-5 py-4">
          <h2 id="cctv-status-heading" className="text-base font-bold">Distribusi Status CCTV</h2>
          <p className="mt-0.5 text-xs text-muted">Sebaran kondisi seluruh perangkat terdaftar.</p>
        </div>

        {statuses.length === 0 ? (
          <p className="px-5 py-8 text-sm text-muted">Belum ada data CCTV.</p>
        ) : (
          <ul className="space-y-3.5 px-5 py-5">
            {statuses.map((item) => {
              const pct = statusTotal === 0 ? 0 : Math.round((item.count / statusTotal) * 100);
              return (
                <li key={item.status}>
                  <div className="mb-1.5 flex items-center justify-between gap-3 text-sm">
                    <span className="font-semibold">{cctvStatusLabel(item.status)}</span>
                    <span className="text-xs font-semibold text-muted tabular-nums">{item.count} ({pct}%)</span>
                  </div>
                  <div className="h-2.5 w-full overflow-hidden rounded-full bg-background">
                    <div
                      className={`h-full rounded-full ${item.status === 'aktif' ? 'bg-success' : item.status === 'rusak' || item.status === 'nonaktif' ? 'bg-error' : 'bg-warning'}`}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>

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
