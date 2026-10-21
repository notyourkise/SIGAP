'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import SortableHeader from '@/components/SortableHeader';
import { useAuth } from '@/context/AuthContext';
import { useSortableRows } from '@/hooks/useSortableRows';
import { apiRequest } from '@/lib/api';
import {
  AlertTriangle,
  LoaderCircle,
  Target,
  BarChart3,
  ClipboardList,
  Trophy,
  ShieldAlert,
} from 'lucide-react';
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Cell,
} from 'recharts';

interface MonthlyCompliance {
  month: string;
  total: number;
  completed_in_target: number;
  compliance_rate: number;
}

interface Violation {
  id: string;
  cctv_name: string;
  cctv_kode: string;
  technician_name: string;
  action_taken: string;
  status: string;
  target_jam_selesai: number;
  deadline: string | null;
  completed_at: string | null;
  overdue_hours: number;
  jenis_tugas: string;
}

interface TechnicianRank {
  user_id: string;
  full_name: string;
  total_tasks: number;
  completed_in_target: number;
  compliance_rate: number;
}

interface SLAResponse {
  monthly_compliance: MonthlyCompliance[];
  violations: Violation[];
  technician_ranking: TechnicianRank[];
}

const JENIS_LABELS: Record<string, string> = {
  corrective: 'Korektif',
  preventive: 'Preventif',
  field_check: 'Pengecekan Lapangan',
};

const STATUS_LABELS: Record<string, string> = {
  scheduled: 'Terjadwal',
  ongoing: 'Berjalan',
  completed: 'Selesai',
  failed: 'Gagal',
};

const chartColors = ['#467235', '#ffbf00', '#b54432', '#3b82f6'];

export default function SLAPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [data, setData] = useState<SLAResponse | null>(null);

  const {
    sortedRows: sortedRanking,
    sortKey: rankingSortKey,
    sortDirection: rankingSortDirection,
    onSort: onRankingSort,
  } = useSortableRows<TechnicianRank>(data?.technician_ranking ?? [], {
    getValue: (row, key) => {
      switch (key) {
        case 'full_name':
          return row.full_name;
        case 'total_tasks':
          return row.total_tasks;
        case 'completed_in_target':
          return row.completed_in_target;
        case 'compliance_rate':
          return row.compliance_rate;
        default:
          return null;
      }
    },
    defaultKey: 'compliance_rate',
    defaultDirection: 'desc',
  });

  const {
    sortedRows: sortedViolations,
    sortKey: violationSortKey,
    sortDirection: violationSortDirection,
    onSort: onViolationSort,
  } = useSortableRows<Violation>(data?.violations ?? [], {
    getValue: (row, key) => {
      switch (key) {
        case 'cctv_name':
          return row.cctv_name;
        case 'jenis_tugas':
          return row.jenis_tugas;
        case 'technician_name':
          return row.technician_name;
        case 'status':
          return row.status;
        case 'deadline':
          return row.deadline;
        case 'overdue_hours':
          return row.overdue_hours;
        default:
          return null;
      }
    },
    defaultKey: 'deadline',
    defaultDirection: 'desc',
  });

  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user]);

  const fetchSLA = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const response = await apiRequest<SLAResponse>('/dashboard/sla?months=6');
      if (response.status === 'success' && response.data) {
        setData(response.data);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Data SLA belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (user?.role !== 'admin') return;
    let cancelled = false;
    const load = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchSLA();
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [fetchSLA, user?.role]);

  const formatMonth = (key: string) => {
    if (!key) return key;
    const [y, m] = key.split('-');
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
    return `${months[Number(m) - 1]} ${y.slice(2)}`;
  };

  const chartData = (data?.monthly_compliance || []).map((m) => ({
    name: formatMonth(m.month),
    total: m.total,
    compliance: Math.round(m.compliance_rate),
  }));

  const avgCompliance = (() => {
    const list = data?.monthly_compliance || [];
    if (list.length === 0) return 0;
    const sum = list.reduce((acc, m) => acc + m.compliance_rate, 0);
    return Math.round(sum / list.length);
  })();

  const totalViolations = data?.violations?.length ?? 0;

  return (
    <DashboardLayout>
      <div className="space-y-6 text-foreground">
        <header>
          <h1 className="text-3xl font-bold tracking-[-0.025em]">SLA Compliance</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted">
            Pantau kepatuhan target waktu layanan (Service Level Agreement) terhadap penyelesaian tugas pemeliharaan.
          </p>
        </header>

        {error && (
          <div role="alert" className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">{error}</div>
        )}

        {loading ? (
          <div className="flex items-center justify-center gap-3 py-24" role="status">
            <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
            <span className="text-sm text-muted">Memuat data SLA…</span>
          </div>
        ) : data ? (
          <>
            {/* Statistik ringkas */}
            <section aria-label="Ringkasan SLA" className="grid overflow-hidden rounded-2xl border border-border bg-card sm:grid-cols-2 lg:grid-cols-3">
              <div className="flex min-h-20 items-center justify-between gap-6 border-b border-border px-5 py-5 lg:border-b-0 lg:border-r">
                <div className="flex items-center gap-2.5">
                  <Target className="h-4 w-4 text-primary" />
                  <span className="text-sm font-medium text-muted">Rata-rata Kepatuhan</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{avgCompliance}%</strong>
              </div>
              <div className="flex min-h-20 items-center justify-between gap-6 border-b border-border px-5 py-5 sm:border-r lg:border-b-0">
                <div className="flex items-center gap-2.5">
                  <ShieldAlert className="h-4 w-4 text-error" />
                  <span className="text-sm font-medium text-muted">Pelanggaran SLA</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums text-error">{totalViolations}</strong>
              </div>
              <div className="flex min-h-20 items-center justify-between gap-6 px-5 py-5">
                <div className="flex items-center gap-2.5">
                  <ClipboardList className="h-4 w-4 text-primary" />
                  <span className="text-sm font-medium text-muted">Teknisi Dinilai</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{data.technician_ranking.length}</strong>
              </div>
            </section>

            {/* Chart compliance bulanan */}
            <section aria-labelledby="sla-chart-title" className="rounded-2xl border border-border bg-card p-5">
              <div className="mb-4 flex items-center gap-2">
                <BarChart3 className="h-5 w-5 text-primary" />
                <h2 id="sla-chart-title" className="font-semibold">Kepatuhan SLA Bulanan</h2>
              </div>
              {chartData.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                  <BarChart3 className="h-10 w-10 text-muted/45" />
                  <p className="mt-3 text-sm text-muted">Belum ada data SLA untuk 6 bulan terakhir.</p>
                </div>
              ) : (
                <div className="h-72 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={chartData} margin={{ top: 10, right: 16, left: -12, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
                      <XAxis
                        dataKey="name"
                        tick={{ fontSize: 12, fill: 'var(--muted-color)' }}
                        stroke="var(--border-color)"
                      />
                      <YAxis
                        domain={[0, 100]}
                        tick={{ fontSize: 12, fill: 'var(--muted-color)' }}
                        stroke="var(--border-color)"
                        unit="%"
                      />
                      <Tooltip
                        formatter={(value) => [`${value ?? 0}%`, 'Kepatuhan']}
                        labelFormatter={(label) => `Bulan: ${label}`}
                        contentStyle={{
                          borderRadius: 12,
                          border: '1px solid var(--border-color)',
                          backgroundColor: 'var(--card-bg)',
                          color: 'var(--foreground)',
                        }}
                        labelStyle={{ color: 'var(--foreground)' }}
                        itemStyle={{ color: 'var(--foreground)' }}
                      />
                      <Bar dataKey="compliance" radius={[8, 8, 0, 0]}>
                        {chartData.map((entry, index) => (
                          <Cell key={entry.name} fill={chartColors[index % chartColors.length]} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              )}
            </section>

            {/* Ranking teknisi */}
            <section aria-labelledby="sla-ranking-title" className="rounded-2xl border border-border bg-card">
              <div className="flex items-center gap-2 border-b border-border px-5 py-4">
                <Trophy className="h-5 w-5 text-warning" />
                <h2 id="sla-ranking-title" className="font-semibold">Ranking Teknisi</h2>
              </div>
              {data.technician_ranking.length === 0 ? (
                <div className="px-6 py-12 text-center text-sm text-muted">Belum ada data ranking teknisi.</div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[640px] border-collapse text-left">
                    <thead>
                      <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                        <th className="px-5 py-3.5">#</th>
                        <th className="px-5 py-3.5"><SortableHeader label="Teknisi" sortKey="full_name" activeKey={rankingSortKey} direction={rankingSortDirection} onSort={onRankingSort} /></th>
                        <th className="px-5 py-3.5"><SortableHeader label="Total Tugas" sortKey="total_tasks" activeKey={rankingSortKey} direction={rankingSortDirection} onSort={onRankingSort} /></th>
                        <th className="px-5 py-3.5"><SortableHeader label="Dalam Target" sortKey="completed_in_target" activeKey={rankingSortKey} direction={rankingSortDirection} onSort={onRankingSort} /></th>
                        <th className="px-5 py-3.5"><SortableHeader label="Kepatuhan" sortKey="compliance_rate" activeKey={rankingSortKey} direction={rankingSortDirection} onSort={onRankingSort} /></th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border text-sm">
                      {sortedRanking.map((rank, idx) => (
                        <tr key={rank.user_id} className="transition-colors hover:bg-background/40">
                          <td className="px-5 py-4 font-bold text-muted">{idx + 1}</td>
                          <td className="px-5 py-4 font-semibold">{rank.full_name || '-'}</td>
                          <td className="px-5 py-4 tabular-nums">{rank.total_tasks}</td>
                          <td className="px-5 py-4 tabular-nums">{rank.completed_in_target}</td>
                          <td className="px-5 py-4">
                            <div className="flex items-center gap-2">
                              <div className="h-2 w-24 overflow-hidden rounded-full bg-background">
                                <div
                                  className={`h-full rounded-full ${rank.compliance_rate >= 70 ? 'bg-success' : rank.compliance_rate >= 40 ? 'bg-warning' : 'bg-error'}`}
                                  style={{ width: `${Math.min(100, rank.compliance_rate)}%` }}
                                />
                              </div>
                              <span className="text-xs font-bold tabular-nums">{Math.round(rank.compliance_rate)}%</span>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>

            {/* Tabel pelanggaran */}
            <section aria-labelledby="sla-violation-title" className="rounded-2xl border border-border bg-card">
              <div className="flex items-center gap-2 border-b border-border px-5 py-4">
                <AlertTriangle className="h-5 w-5 text-error" />
                <h2 id="sla-violation-title" className="font-semibold">Tabel Pelanggaran SLA</h2>
                <span className="ml-auto text-xs text-muted">{totalViolations} pelanggaran</span>
              </div>
              {data.violations.length === 0 ? (
                <div className="px-6 py-12 text-center text-sm text-muted">Tidak ada pelanggaran SLA. Semua tugas selesai dalam target.</div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[900px] border-collapse text-left">
                    <thead>
                      <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                        <th className="px-5 py-3.5"><SortableHeader label="CCTV" sortKey="cctv_name" activeKey={violationSortKey} direction={violationSortDirection} onSort={onViolationSort} /></th>
                        <th className="px-5 py-3.5"><SortableHeader label="Jenis" sortKey="jenis_tugas" activeKey={violationSortKey} direction={violationSortDirection} onSort={onViolationSort} /></th>
                        <th className="px-5 py-3.5"><SortableHeader label="Teknisi" sortKey="technician_name" activeKey={violationSortKey} direction={violationSortDirection} onSort={onViolationSort} /></th>
                        <th className="px-5 py-3.5"><SortableHeader label="Status" sortKey="status" activeKey={violationSortKey} direction={violationSortDirection} onSort={onViolationSort} /></th>
                        <th className="px-5 py-3.5"><SortableHeader label="Deadline" sortKey="deadline" activeKey={violationSortKey} direction={violationSortDirection} onSort={onViolationSort} /></th>
                        <th className="px-5 py-3.5 text-right"><SortableHeader label="Overdue" sortKey="overdue_hours" activeKey={violationSortKey} direction={violationSortDirection} onSort={onViolationSort} align="right" /></th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border text-sm">
                      {sortedViolations.map((v) => (
                        <tr key={v.id} className="transition-colors hover:bg-background/40">
                          <td className="px-5 py-4">
                            <p className="font-semibold">{v.cctv_name || '-'}</p>
                            <p className="font-mono text-xs text-primary">{v.cctv_kode}</p>
                          </td>
                          <td className="px-5 py-4 text-muted">{JENIS_LABELS[v.jenis_tugas] || v.jenis_tugas || '-'}</td>
                          <td className="px-5 py-4">{v.technician_name || '-'}</td>
                          <td className="px-5 py-4">
                            <span className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold ${
                              v.status === 'completed'
                                ? 'border-success/25 bg-success/10 text-success'
                                : v.status === 'ongoing'
                                  ? 'border-warning/25 bg-warning/10 text-warning'
                                  : 'border-blue-500/25 bg-blue-500/10 text-blue-600 dark:text-blue-300'
                            }`}>
                              {STATUS_LABELS[v.status] || v.status}
                            </span>
                          </td>
                          <td className="px-5 py-4 text-xs text-muted">
                            <div>{v.deadline || '-'}</div>
                            {v.completed_at && <div className="mt-0.5">Selesai: {v.completed_at}</div>}
                          </td>
                          <td className="px-5 py-4 text-right">
                            <span className="font-bold tabular-nums text-error">{v.overdue_hours} jam</span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          </>
        ) : (
          <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
            <AlertTriangle className="h-10 w-10 text-error" />
            <h3 className="mt-3 text-base font-semibold">Data SLA belum dapat dimuat</h3>
            <button
              type="button"
              onClick={fetchSLA}
              className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover"
            >
              Coba lagi
            </button>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}
