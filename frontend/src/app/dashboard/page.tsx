'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import DashboardLayout from '@/components/DashboardLayout';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';
import {
  Activity,
  Camera,
  CheckCircle2,
  Clock3,
  Database,
  QrCode,
  ShieldCheck,
  Users,
  Wrench,
} from 'lucide-react';

interface CCTV {
  id: string;
  status: string;
}

interface PaginationPayload<T> {
  rows: T[];
  pagination: {
    total_rows: number;
  };
}

interface DashboardStats {
  total_maintenance_logs: number;
  logs_this_month: number;
  logs_completed_this_month: number;
  logs_pending_this_month: number;
  total_users: number;
}

const normalizeStatus = (status: string) => status.trim().toUpperCase();

export default function DashboardPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const [cctvs, setCctvs] = useState<CCTV[]>([]);
  const [totalCctv, setTotalCctv] = useState(0);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [health, setHealth] = useState<'Memeriksa' | 'Terhubung' | 'Tidak terhubung'>('Memeriksa');
  const [error, setError] = useState('');

  const loadDashboard = useCallback(async () => {
    setError('');
    try {
      const [cctvResponse, statsResponse] = await Promise.all([
        apiRequest<PaginationPayload<CCTV>>('/cctv?page=1&limit=100'),
        apiRequest<DashboardStats>('/dashboard/stats'),
        apiRequest<Record<string, unknown>>('/health'),
      ]);

      const cctvData = cctvResponse.data;
      const rows = Array.isArray(cctvData) ? cctvData : (cctvData?.rows || []);
      setCctvs(rows);
      setTotalCctv(cctvData?.pagination?.total_rows ?? rows.length);
      setStats(statsResponse.data || null);
      setHealth('Terhubung');
    } catch (err) {
      setHealth('Tidak terhubung');
      setError(err instanceof Error ? err.message : 'Dashboard tidak dapat dimuat');
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(loadDashboard, 0);
    return () => clearTimeout(timer);
  }, [loadDashboard]);

  const active = cctvs.filter((item) => ['ACTIVE', 'AKTIF'].includes(normalizeStatus(item.status))).length;
  const inactive = cctvs.filter((item) => ['INACTIVE', 'NONAKTIF', 'BROKEN', 'RUSAK'].includes(normalizeStatus(item.status))).length;
  const maintenance = cctvs.filter((item) => ['MAINTENANCE', 'PEMELIHARAAN'].includes(normalizeStatus(item.status))).length;

  const adminCards = [
    { label: 'Total CCTV', value: totalCctv || '—', icon: Camera, color: 'bg-blue-500/10 text-blue-600' },
    { label: 'CCTV Aktif', value: active, icon: CheckCircle2, color: 'bg-success/10 text-success' },
    { label: 'CCTV Nonaktif', value: inactive, icon: Activity, color: 'bg-error/10 text-error' },
    { label: 'Pengguna', value: stats?.total_users ?? '—', icon: Users, color: 'bg-primary/10 text-primary' },
  ];

  const technicianCards = [
    { label: 'CCTV Terpantau', value: totalCctv || '—', icon: Camera, color: 'bg-blue-500/10 text-blue-600' },
    { label: 'CCTV Aktif', value: active, icon: CheckCircle2, color: 'bg-success/10 text-success' },
    { label: 'Perlu Ditangani', value: inactive + maintenance, icon: Wrench, color: 'bg-warning/10 text-warning' },
    { label: 'Log Bulan Ini', value: stats?.logs_this_month ?? '—', icon: Clock3, color: 'bg-primary/10 text-primary' },
  ];

  const cards = isAdmin ? adminCards : technicianCards;

  return (
    <DashboardLayout>
      <div className="space-y-7">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="mb-2 flex items-center gap-2">
              <span className={`rounded-full px-3 py-1 text-xs font-bold uppercase tracking-wider ${isAdmin ? 'bg-warning/10 text-warning' : 'bg-success/10 text-success'}`}>
                {isAdmin ? 'Dashboard Admin' : 'Dashboard Teknisi'}
              </span>
            </div>
            <h1 className="text-3xl font-bold tracking-tight">
              Selamat datang, {user?.full_name || (isAdmin ? 'Admin' : 'Teknisi')}
            </h1>
            {isAdmin && (
              <p className="mt-1 text-sm text-muted">
                Ringkasan pengelolaan aset, pemeliharaan, dan pengguna SIGAP.
              </p>
            )}
          </div>
          <div className="inline-flex w-fit items-center gap-2 rounded-full border border-border bg-card px-3 py-2 text-xs font-semibold">
            <span className={`h-2 w-2 rounded-full ${health === 'Terhubung' ? 'bg-success' : health === 'Memeriksa' ? 'bg-warning' : 'bg-error'}`} />
            API {health}
          </div>
        </div>

        {error && <div className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">{error}</div>}

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {cards.map(({ label, value, icon: Icon, color }) => (
            <div key={label} className="rounded-2xl border border-border bg-card p-5 shadow-sm">
              <div className={`mb-4 flex h-10 w-10 items-center justify-center rounded-xl ${color}`}>
                <Icon className="h-5 w-5" />
              </div>
              <p className="text-xs font-semibold uppercase tracking-wider text-muted">{label}</p>
              <p className="mt-1 text-3xl font-bold">{value}</p>
            </div>
          ))}
        </div>

        {isAdmin ? (
          <AdminContent stats={stats} maintenanceCount={maintenance} />
        ) : (
          <TechnicianContent
            inactiveCount={inactive}
            maintenanceCount={maintenance}
            completedThisMonth={stats?.logs_completed_this_month || 0}
            pendingThisMonth={stats?.logs_pending_this_month || 0}
          />
        )}
      </div>
    </DashboardLayout>
  );
}

function AdminContent({ stats, maintenanceCount }: { stats: DashboardStats | null; maintenanceCount: number }) {
  return (
    <div className="grid gap-5 lg:grid-cols-[1fr_1.15fr]">
      <section className="rounded-2xl border border-border bg-card p-5">
        <h2 className="font-semibold">Aktivitas Pemeliharaan</h2>
        <div className="mt-5 space-y-3">
          <SummaryRow label="Total log" value={stats?.total_maintenance_logs ?? 0} icon={Wrench} />
          <SummaryRow label="Log bulan ini" value={stats?.logs_this_month ?? 0} icon={Clock3} />
          <SummaryRow label="CCTV berstatus pemeliharaan" value={maintenanceCount} icon={Activity} />
        </div>
      </section>
      <section className="rounded-2xl border border-border bg-card p-5">
        <h2 className="font-semibold">Akses Pengelolaan Admin</h2>
        <p className="mt-1 text-xs text-muted">Menu dengan kewenangan tambah, ubah, dan hapus data.</p>
        <div className="mt-5 grid gap-3 sm:grid-cols-3">
          <QuickLink href="/cctv" label="Kelola CCTV" icon={Database} />
          <QuickLink href="/maintenance" label="Pemeliharaan" icon={Wrench} />
          <QuickLink href="/users" label="Pengguna" icon={Users} />
        </div>
        <div className="mt-5 flex items-start gap-3 rounded-xl border border-primary/15 bg-primary/5 p-4">
          <ShieldCheck className="mt-0.5 h-5 w-5 shrink-0 text-primary" />
          <p className="text-xs leading-relaxed text-muted">
            Admin dapat mengelola data CCTV, impor Excel, bulk delete, menghapus log, dan mengelola akun admin maupun teknisi.
          </p>
        </div>
      </section>
    </div>
  );
}

function TechnicianContent({
  inactiveCount,
  maintenanceCount,
  completedThisMonth,
  pendingThisMonth,
}: {
  inactiveCount: number;
  maintenanceCount: number;
  completedThisMonth: number;
  pendingThisMonth: number;
}) {
  return (
    <div className="grid gap-5 lg:grid-cols-[1fr_1.15fr]">
      <section className="rounded-2xl border border-border bg-card p-5">
        <h2 className="font-semibold">Prioritas Teknisi</h2>
        <div className="mt-5 space-y-3">
          <SummaryRow label="CCTV nonaktif" value={inactiveCount} icon={Activity} />
          <SummaryRow label="Dalam pemeliharaan" value={maintenanceCount} icon={Wrench} />
          <SummaryRow label="Log menunggu bulan ini" value={pendingThisMonth} icon={Clock3} />
          <SummaryRow label="Selesai bulan ini" value={completedThisMonth} icon={CheckCircle2} />
        </div>
      </section>
      <section className="rounded-2xl border border-border bg-card p-5">
        <h2 className="font-semibold">Akses Operasional Teknisi</h2>
        <p className="mt-1 text-xs text-muted">Aksi yang tersedia sesuai hak akses teknisi.</p>
        <div className="mt-5 grid gap-3 sm:grid-cols-3">
          <QuickLink href="/cctv" label="Lihat CCTV" icon={Camera} />
          <QuickLink href="/cctv" label="Scanner QR" icon={QrCode} />
          <QuickLink href="/maintenance" label="Catat Perawatan" icon={Wrench} />
        </div>
        <div className="mt-5 flex items-start gap-3 rounded-xl border border-success/15 bg-success/5 p-4">
          <ShieldCheck className="mt-0.5 h-5 w-5 shrink-0 text-success" />
          <p className="text-xs leading-relaxed text-muted">
            Teknisi dapat melihat dan memindai CCTV, membuat atau memperbarui catatan pemeliharaan, mengunggah bukti foto, serta mengekspor riwayat. Pengelolaan master, pengguna, impor, dan penghapusan tidak ditampilkan.
          </p>
        </div>
      </section>
    </div>
  );
}

function SummaryRow({ label, value, icon: Icon }: { label: string; value: number; icon: typeof Activity }) {
  return (
    <div className="flex items-center justify-between rounded-xl bg-background px-4 py-3">
      <span className="flex items-center gap-3 text-sm text-muted"><Icon className="h-4 w-4 text-primary" /> {label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function QuickLink({ href, label, icon: Icon }: { href: string; label: string; icon: typeof Activity }) {
  return (
    <Link href={href} className="flex flex-col items-center gap-2 rounded-xl border border-border bg-background p-4 text-center text-xs font-semibold transition-colors hover:border-primary hover:text-primary">
      <Icon className="h-5 w-5" />
      {label}
    </Link>
  );
}
