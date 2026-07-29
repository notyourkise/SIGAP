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
        setError(cctvResult.reason instanceof Error ? cctvResult.reason.message : 'Data CCTV tidak dapat dimuat.');
      }

      if (maintenanceResult.status === 'fulfilled') {
        const maintenanceData = maintenanceResult.value.data;
        const maintenanceRows = maintenanceData?.rows || [];
        setLogs(maintenanceRows);
        setTotalLogs(maintenanceData?.pagination?.total_rows ?? maintenanceRows.length);
      } else {
        setError((current) => current || (maintenanceResult.reason instanceof Error ? maintenanceResult.reason.message : 'Data pemeliharaan tidak dapat dimuat.'));
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

  const adminCards = [
    { label: 'Total CCTV', value: totalCctv || '—', tone: 'border-t-blue-500' },
    { label: 'CCTV Aktif', value: active, tone: 'border-t-success' },
    { label: 'CCTV Nonaktif', value: inactive, tone: 'border-t-error' },
    { label: 'Pemeliharaan', value: totalLogs, tone: 'border-t-warning' },
  ];

  const technicianCards = [
    { label: 'Total CCTV', value: totalCctv || '—', tone: 'border-t-blue-500' },
    { label: 'CCTV Aktif', value: active, tone: 'border-t-success' },
    { label: 'CCTV Nonaktif', value: inactive, tone: 'border-t-error' },
    { label: 'Pemeliharaan', value: totalLogs, tone: 'border-t-warning' },
  ];

  const cards = isAdmin ? adminCards : technicianCards;
  const healthDescription = health === 'Terhubung'
    ? 'Backend merespons normal dan data dashboard berhasil dimuat.'
    : health === 'Memeriksa'
      ? 'Sistem sedang memeriksa koneksi ke backend.'
      : 'Backend tidak dapat dijangkau. Periksa service dan port API.';

  return (
    <DashboardLayout>
      <div className="space-y-7">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <span className={`inline-flex rounded-full px-3 py-1 text-xs font-bold uppercase tracking-wider ${isAdmin ? 'bg-warning/10 text-warning' : 'bg-success/10 text-success'}`}>
              {isAdmin ? 'Dashboard Admin' : 'Dashboard Teknisi'}
            </span>
            <h1 className="mt-3 text-3xl font-bold tracking-tight">
              Selamat datang, {user?.full_name || (isAdmin ? 'Admin' : 'Teknisi')}
            </h1>
            {isAdmin && <p className="mt-1 text-sm text-muted">Ringkasan pengelolaan aset, pemeliharaan, dan pengguna SIGAP.</p>}
          </div>
          <button
            type="button"
            onClick={loadDashboard}
            disabled={loading}
            className="inline-flex w-fit items-center gap-2 rounded-full border border-border bg-card px-3 py-2 text-xs font-semibold transition-colors hover:border-primary disabled:cursor-wait disabled:opacity-70"
          >
            <span className={`h-2 w-2 rounded-full ${health === 'Terhubung' ? 'bg-success' : health === 'Memeriksa' ? 'animate-pulse bg-warning' : 'bg-error'}`} />
            API {health}
          </button>
        </div>

        {error && <div className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">{error}</div>}

        {loading ? (
          <DashboardSkeleton />
        ) : (
          <>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
              {cards.map(({ label, value, tone }) => (
                <div key={label} className={`rounded-2xl border border-t-4 border-border bg-card p-5 shadow-sm ${tone}`}>
                  <p className="text-xs font-semibold uppercase tracking-wider text-muted">{label}</p>
                  <p className="mt-1 text-3xl font-bold">{value}</p>
                </div>
              ))}
            </div>

            <div className="grid gap-5 lg:grid-cols-[1.45fr_0.75fr]">
              <section className="rounded-2xl border border-border bg-card p-5">
                <h2 className="font-semibold">{isAdmin ? 'Aktivitas Pemeliharaan' : 'Prioritas Teknisi'}</h2>
                <div className="mt-5 grid gap-3 sm:grid-cols-2">
                  <StatusCard label="Terjadwal" value={scheduled} note="Belum mulai dikerjakan" tone="blue" active={selectedStatus === 'scheduled'} onClick={() => setSelectedStatus('scheduled')} />
                  <StatusCard label="Dalam proses" value={ongoing} note="Sedang dikerjakan" tone="warning" active={selectedStatus === 'ongoing'} onClick={() => setSelectedStatus('ongoing')} />
                  <StatusCard label="Selesai" value={completed} note="Pekerjaan telah dituntaskan" tone="success" active={selectedStatus === 'completed'} onClick={() => setSelectedStatus('completed')} />
                  <StatusCard label="Gagal" value={failed} note="Memerlukan tindak lanjut" tone="error" active={selectedStatus === 'failed'} onClick={() => setSelectedStatus('failed')} />
                </div>
                {selectedStatus && (
                  <div className="mt-5 border-t border-border pt-5">
                    <div className="mb-3 flex items-center justify-between">
                      <h3 className="text-sm font-semibold">Daftar CCTV — {statusLabel(selectedStatus)}</h3>
                      <button type="button" onClick={() => setSelectedStatus(null)} className="text-xs font-semibold text-muted hover:text-foreground">Tutup</button>
                    </div>
                    {selectedLogs.length === 0 ? (
                      <p className="rounded-xl bg-background px-4 py-4 text-sm text-muted">Tidak ada pekerjaan dengan status ini.</p>
                    ) : (
                      <div className="space-y-2">
                        {selectedLogs.map((log) => (
                          <div key={log.id} className="grid gap-2 rounded-xl border border-border bg-background px-4 py-3 sm:grid-cols-[1fr_1fr]">
                            <div><p className="text-xs text-muted">CCTV</p><p className="mt-0.5 text-sm font-semibold">{log.asset?.name || 'CCTV tidak tersedia'}</p></div>
                            <div><p className="text-xs text-muted">Lokasi</p><p className="mt-0.5 text-sm font-medium">{log.asset?.location_name || 'Lokasi belum dicatat'}</p></div>
                            <p className="text-xs text-muted sm:col-span-2">{log.action_taken}</p>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </section>

              <section className={`overflow-hidden rounded-2xl border bg-card ${health === 'Terhubung' ? 'border-success/35' : 'border-error/35'}`}>
                <div className={`h-1.5 ${health === 'Terhubung' ? 'bg-success' : 'bg-error'}`} />
                <div className="p-5">
                  <p className="text-xs font-semibold uppercase tracking-wider text-muted">Kesehatan API</p>
                  <div className="mt-3 flex items-center justify-between gap-3">
                    <div className="flex items-center gap-2">
                      <span className={`h-2.5 w-2.5 rounded-full ${health === 'Terhubung' ? 'bg-success' : 'bg-error'}`} />
                      <h2 className="text-lg font-bold">{health === 'Terhubung' ? 'Normal' : health}</h2>
                    </div>
                    <span className={`rounded-full px-2.5 py-1 text-[11px] font-bold uppercase ${health === 'Terhubung' ? 'bg-success/10 text-success' : 'bg-error/10 text-error'}`}>
                      {health === 'Terhubung' ? 'Online' : 'Offline'}
                    </span>
                  </div>
                  <p className="mt-4 text-sm leading-relaxed text-muted">{healthDescription}</p>
                  <dl className="mt-5 divide-y divide-border rounded-xl border border-border bg-background px-4">
                    <div className="flex items-center justify-between py-3 text-xs"><dt className="text-muted">Layanan backend</dt><dd className="font-semibold">{health === 'Terhubung' ? 'Tersedia' : 'Tidak tersedia'}</dd></div>
                    <div className="flex items-center justify-between py-3 text-xs"><dt className="text-muted">Sinkronisasi data</dt><dd className="font-semibold">{health === 'Terhubung' ? 'Aktif' : 'Terhenti'}</dd></div>
                    <div className="flex items-center justify-between py-3 text-xs"><dt className="text-muted">Diperiksa</dt><dd className="font-semibold">{new Date().toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })}</dd></div>
                  </dl>
                </div>
              </section>
            </div>
          </>
        )}
      </div>
    </DashboardLayout>
  );
}

function StatusCard({ label, value, note, tone, active, onClick }: {
  label: string;
  value: number;
  note: string;
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
      className={`flex w-full items-center gap-4 rounded-xl border bg-background p-4 text-left transition-colors hover:border-primary ${active ? 'border-primary ring-2 ring-primary/10' : 'border-border'}`}
    >
      <span className={`h-2.5 w-2.5 shrink-0 rounded-full ${tones[tone]}`} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-3"><p className="text-sm font-semibold">{label}</p><strong className="text-xl">{value}</strong></div>
        <p className="mt-0.5 text-xs text-muted">{note}</p>
      </div>
    </button>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-5" aria-label="Memuat dashboard">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => <div key={index} className="h-36 animate-pulse rounded-2xl border border-border bg-card" />)}
      </div>
      <div className="grid gap-5 lg:grid-cols-[1.45fr_0.75fr]">
        <div className="h-80 animate-pulse rounded-2xl border border-border bg-card" />
        <div className="h-80 animate-pulse rounded-2xl border border-border bg-card" />
      </div>
    </div>
  );
}
