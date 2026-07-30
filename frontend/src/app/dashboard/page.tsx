'use client';

import { useCallback, useEffect, useState } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';

interface CCTV {
  id: string;
  status: string;
}

interface MaintenanceLog {
  id: string;
  status: 'scheduled' | 'ongoing' | 'completed' | 'failed';
  action_taken: string;
  created_at: string;
  asset?: {
    name?: string;
    location_name?: string;
  };
}

interface PaginationPayload<T> {
  rows: T[];
  pagination: {
    total_rows: number;
  };
}

const normalizeStatus = (status: string) => status.trim().toUpperCase();
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
  const [selectedStatus, setSelectedStatus] = useState<MaintenanceLog['status'] | null>(null);
  const [health, setHealth] = useState<'Memeriksa' | 'Terhubung' | 'Tidak terhubung'>('Memeriksa');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadDashboard = useCallback(async () => {
    setLoading(true);
    setError('');
    setHealth('Memeriksa');
    try {
      const [cctvResult, maintenanceResult, healthResult] = await Promise.allSettled([
        apiRequest<PaginationPayload<CCTV>>('/cctv?page=1&limit=100'),
        apiRequest<PaginationPayload<MaintenanceLog>>('/maintenance-logs?page=1&limit=100'),
        apiRequest<Record<string, unknown>>('/health'),
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

      setHealth(healthResult.status === 'fulfilled' ? 'Terhubung' : 'Tidak terhubung');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(loadDashboard, 0);
    return () => clearTimeout(timer);
  }, [loadDashboard]);

  const active = cctvs.filter((item) => ['ACTIVE', 'AKTIF'].includes(normalizeStatus(item.status))).length;
  const inactive = cctvs.filter((item) => ['INACTIVE', 'NONAKTIF', 'BROKEN', 'RUSAK'].includes(normalizeStatus(item.status))).length;
  const scheduled = logs.filter((item) => item.status === 'scheduled').length;
  const ongoing = logs.filter((item) => item.status === 'ongoing').length;
  const completed = logs.filter((item) => item.status === 'completed').length;
  const failed = logs.filter((item) => item.status === 'failed').length;
  const selectedLogs = selectedStatus ? logs.filter((item) => item.status === selectedStatus) : [];

  const summary = [
    { label: 'Total CCTV', value: totalCctv, tone: 'bg-blue-500' },
    { label: 'Aktif', value: active, tone: 'bg-success' },
    { label: 'Nonaktif', value: inactive, tone: 'bg-error' },
  ];

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
                {summary.map(({ label, value, tone }) => (
                  <div key={label} className="flex items-center justify-between gap-6 px-5 py-5">
                    <div className="flex items-center gap-2.5">
                      <span className={`h-2.5 w-2.5 rounded-full ${tone}`} />
                      <p className="text-sm font-medium text-muted">{label}</p>
                    </div>
                    <p className="text-2xl font-bold tabular-nums">{value}</p>
                  </div>
                ))}
              </div>
            </section>

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
                            <p className="text-sm font-semibold">{log.asset?.name || 'CCTV tidak tersedia'}</p>
                            <p className="mt-0.5 text-xs text-muted">{log.asset?.location_name || 'Lokasi belum dicatat'}</p>
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
